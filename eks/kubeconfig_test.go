package eks

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"
)

const sampleKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: QUJDREVG
    server: https://40FE8DD75AA0D244738B66619C01CC9E.yl4.eu-west-2.eks.amazonaws.com
  name: arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-external
- cluster:
    certificate-authority-data: WFlaWFla
    server: https://84225D38EAC43B59B45AB872E558383D.gr7.eu-west-2.eks.amazonaws.com
  name: arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-internal
contexts:
- context:
    cluster: arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-external
    user: dis-bleed-external-view-only
  name: dis-bleed-external
- context:
    cluster: arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-internal
    user: dis-bleed-internal-view-only
  name: dis-bleed-internal
current-context: dis-bleed-internal
users:
- name: dis-bleed-external-view-only
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
`

// findCluster returns the cluster block MapSlice for the cluster entry whose
// top-level name matches ref (an ARN in the AWS-written kubeconfig).
func findCluster(t *testing.T, path, ref string) yaml.MapSlice {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test-controlled path
	So(err, ShouldBeNil)
	var doc yaml.MapSlice
	So(yaml.Unmarshal(data, &doc), ShouldBeNil)
	for _, top := range doc {
		if k, _ := top.Key.(string); k != "clusters" {
			continue
		}
		clusters, _ := top.Value.([]interface{})
		for _, c := range clusters {
			entry, _ := c.(yaml.MapSlice)
			if getMapSliceString(entry, "name") == ref {
				v, _ := getMapSliceValue(entry, "cluster")
				cm, _ := v.(yaml.MapSlice)
				return cm
			}
		}
	}
	return nil
}

func TestKubeconfigPath(t *testing.T) {
	Convey("Given the KUBECONFIG env var", t, func() {
		orig, had := os.LookupEnv("KUBECONFIG")
		defer func() {
			if had {
				_ = os.Setenv("KUBECONFIG", orig)
			} else {
				_ = os.Unsetenv("KUBECONFIG")
			}
		}()

		Convey("When KUBECONFIG is set to a single path", func() {
			So(os.Setenv("KUBECONFIG", "/tmp/custom/config"), ShouldBeNil)
			p, err := KubeconfigPath()
			So(err, ShouldBeNil)
			So(p, ShouldEqual, "/tmp/custom/config")
		})

		Convey("When KUBECONFIG contains multiple colon-separated paths", func() {
			So(os.Setenv("KUBECONFIG", "/tmp/first:/tmp/second"), ShouldBeNil)
			p, err := KubeconfigPath()
			So(err, ShouldBeNil)
			So(p, ShouldEqual, "/tmp/first")
		})

		Convey("When KUBECONFIG is unset, it falls back to ~/.kube/config", func() {
			So(os.Unsetenv("KUBECONFIG"), ShouldBeNil)
			p, err := KubeconfigPath()
			So(err, ShouldBeNil)
			home, _ := os.UserHomeDir()
			So(p, ShouldEqual, filepath.Join(home, ".kube", "config"))
		})
	})
}

func TestPatchKubeconfigForNoSudo(t *testing.T) {
	Convey("Given an AWS-written kubeconfig (ARN-named cluster entries, short context aliases)", t, func() {
		dir := t.TempDir()
		path := filepath.Join(dir, "config")
		So(os.WriteFile(path, []byte(sampleKubeconfig), 0600), ShouldBeNil)
		So(os.Setenv("KUBECONFIG", path), ShouldBeNil)
		defer func() { _ = os.Unsetenv("KUBECONFIG") }()

		endpoint := "40FE8DD75AA0D244738B66619C01CC9E.yl4.eu-west-2.eks.amazonaws.com"
		externalARN := "arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-external"
		internalARN := "arn:aws:eks:eu-west-2:697746439640:cluster/dis-bleed-internal"

		Convey("When patching by the short cluster name (the DescribeCluster name)", func() {
			// This is the exact case that previously failed: the caller passes the
			// short name but the cluster entry is named by ARN. Resolution via the
			// context must find the right entry.
			err := PatchKubeconfigForNoSudo("dis-bleed-external", endpoint, 9443)
			So(err, ShouldBeNil)

			Convey("Then server points at the local port and tls-server-name is the endpoint", func() {
				cm := findCluster(t, path, externalARN)
				So(cm, ShouldNotBeNil)
				So(getMapSliceString(cm, "server"), ShouldEqual, "https://127.0.0.1:9443")
				So(getMapSliceString(cm, "tls-server-name"), ShouldEqual, endpoint)
			})

			Convey("Then certificate-authority-data is preserved", func() {
				cm := findCluster(t, path, externalARN)
				So(getMapSliceString(cm, "certificate-authority-data"), ShouldEqual, "QUJDREVG")
			})

			Convey("Then no insecure-skip-tls-verify is introduced", func() {
				data, _ := os.ReadFile(path)
				So(string(data), ShouldNotContainSubstring, "insecure-skip-tls-verify")
			})

			Convey("Then the other cluster is left untouched", func() {
				cm := findCluster(t, path, internalARN)
				So(getMapSliceString(cm, "server"), ShouldEqual, "https://84225D38EAC43B59B45AB872E558383D.gr7.eu-west-2.eks.amazonaws.com")
				So(getMapSliceString(cm, "tls-server-name"), ShouldEqual, "")
			})

			Convey("Then non-cluster sections (contexts, users) are preserved", func() {
				data, _ := os.ReadFile(path)
				So(string(data), ShouldContainSubstring, "current-context: dis-bleed-internal")
				So(string(data), ShouldContainSubstring, "dis-bleed-external-view-only")
			})
		})

		Convey("When patching is applied twice (idempotency)", func() {
			So(PatchKubeconfigForNoSudo("dis-bleed-external", endpoint, 9443), ShouldBeNil)
			So(PatchKubeconfigForNoSudo("dis-bleed-external", endpoint, 9444), ShouldBeNil)

			Convey("Then the latest values win and no duplicate keys appear", func() {
				cm := findCluster(t, path, externalARN)
				So(getMapSliceString(cm, "server"), ShouldEqual, "https://127.0.0.1:9444")
				// exactly one server and one tls-server-name key
				serverCount, sniCount := 0, 0
				for _, item := range cm {
					switch item.Key {
					case "server":
						serverCount++
					case "tls-server-name":
						sniCount++
					}
				}
				So(serverCount, ShouldEqual, 1)
				So(sniCount, ShouldEqual, 1)
			})
		})

		Convey("When patching a cluster that does not exist", func() {
			err := PatchKubeconfigForNoSudo("no-such-cluster", endpoint, 9443)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "not found")
		})
	})
}
