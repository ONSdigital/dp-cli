package eks

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// withTempTunnelDir points tunnelDir at a temp dir for the duration of a test.
func withTempTunnelDir(t *testing.T) {
	t.Helper()
	orig := tunnelDir
	tunnelDir = t.TempDir()
	t.Cleanup(func() { tunnelDir = orig })
}

func TestTunnelStatePersistence(t *testing.T) {
	Convey("Given a no-sudo tunnel state", t, func() {
		withTempTunnelDir(t)

		state := TunnelState{
			ClusterName: "dis-bleed-external",
			SSMPid:      12345,
			Endpoint:    "40FE8DD75AA0D244738B66619C01CC9E.yl4.eu-west-2.eks.amazonaws.com",
			IPv4:        "10.0.0.5",
			LocalPort:   9445,
			NoSudo:      true,
		}

		Convey("When saved and reloaded", func() {
			So(SaveTunnelState(state), ShouldBeNil)

			Convey("Then it is stored as a single .json file", func() {
				matches, _ := filepath.Glob(filepath.Join(tunnelDir, "*"))
				So(len(matches), ShouldEqual, 1)
				So(matches[0], ShouldEndWith, "dis-bleed-external.json")
			})

			Convey("Then the reloaded state matches", func() {
				loaded, err := LoadTunnelState("dis-bleed-external")
				So(err, ShouldBeNil)
				So(*loaded, ShouldResemble, state)
			})

			Convey("Then ListActiveTunnels returns it", func() {
				tunnels, err := ListActiveTunnels()
				So(err, ShouldBeNil)
				So(len(tunnels), ShouldEqual, 1)
				So(tunnels[0].ClusterName, ShouldEqual, "dis-bleed-external")
				So(tunnels[0].NoSudo, ShouldBeTrue)
			})

			Convey("Then no temp file is left behind", func() {
				matches, _ := filepath.Glob(filepath.Join(tunnelDir, "*.tmp"))
				So(len(matches), ShouldEqual, 0)
			})
		})

		Convey("When cleaned up", func() {
			So(SaveTunnelState(state), ShouldBeNil)
			CleanupTunnelState("dis-bleed-external")

			Convey("Then the state file is removed", func() {
				_, err := LoadTunnelState("dis-bleed-external")
				So(os.IsNotExist(err), ShouldBeTrue)
			})

			Convey("Then ListActiveTunnels is empty", func() {
				tunnels, _ := ListActiveTunnels()
				So(len(tunnels), ShouldEqual, 0)
			})
		})
	})

	Convey("Given leftover legacy per-field fragment files from an older version", t, func() {
		withTempTunnelDir(t)
		prefix := filepath.Join(tunnelDir, "old-cluster")
		for _, ext := range []string{".ssm.pid", ".socat.pid", ".endpoint", ".loopback", ".ipv4", ".port", ".nosudo"} {
			So(os.WriteFile(prefix+ext, []byte("x"), 0600), ShouldBeNil)
		}

		Convey("When CleanupTunnelState runs for that cluster", func() {
			CleanupTunnelState("old-cluster")

			Convey("Then all legacy fragment files are swept", func() {
				matches, _ := filepath.Glob(filepath.Join(tunnelDir, "old-cluster*"))
				So(len(matches), ShouldEqual, 0)
			})
		})

		Convey("Then legacy fragments are not treated as active tunnels", func() {
			// ListActiveTunnels only globs *.json now, so fragments are ignored.
			tunnels, _ := ListActiveTunnels()
			So(len(tunnels), ShouldEqual, 0)
		})
	})
}

func TestPruneTunnelDir(t *testing.T) {
	Convey("Given a tunnel dir with a mix of current and stray files", t, func() {
		withTempTunnelDir(t)

		// An active no-sudo tunnel on port 9445.
		active := TunnelState{ClusterName: "dis-bleed-external", SSMPid: 111, LocalPort: 9445, NoSudo: true}
		So(SaveTunnelState(active), ShouldBeNil)

		// A log referenced by the active tunnel (keep) and an orphaned log (drop).
		So(os.WriteFile(filepath.Join(tunnelDir, "ssm-9445.log"), []byte("x"), 0600), ShouldBeNil)
		So(os.WriteFile(filepath.Join(tunnelDir, "ssm-9999.log"), []byte("x"), 0600), ShouldBeNil)

		// Legacy per-field fragment files (drop).
		for _, ext := range []string{".ssm.pid", ".endpoint", ".loopback", ".ipv4", ".port", ".nosudo", ".socat.pid"} {
			So(os.WriteFile(filepath.Join(tunnelDir, "old-cluster"+ext), []byte("x"), 0600), ShouldBeNil)
		}

		// A stray temp file from an interrupted atomic write (drop).
		So(os.WriteFile(filepath.Join(tunnelDir, "dis-bleed-external.json.tmp"), []byte("x"), 0600), ShouldBeNil)

		Convey("When PruneTunnelDir runs", func() {
			PruneTunnelDir()

			remaining := func() []string {
				entries, _ := os.ReadDir(tunnelDir)
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				return names
			}()

			Convey("Then only the current-format files remain", func() {
				So(remaining, ShouldContain, "dis-bleed-external.json")
				So(remaining, ShouldContain, "ssm-9445.log")
				So(len(remaining), ShouldEqual, 2)
			})

			Convey("Then the active tunnel is still loadable", func() {
				loaded, err := LoadTunnelState("dis-bleed-external")
				So(err, ShouldBeNil)
				So(loaded.LocalPort, ShouldEqual, 9445)
			})
		})
	})
}

func TestConfigure(t *testing.T) {
	Convey("Given Configure with saved originals restored after", t, func() {
		origDir, origBase, origMax := tunnelDir, basePort, maxPort
		t.Cleanup(func() { tunnelDir, basePort, maxPort = origDir, origBase, origMax })

		Convey("When all values are empty/zero, defaults are kept", func() {
			tunnelDir, basePort, maxPort = DefaultStateDir, DefaultBasePort, DefaultMaxPort
			Configure("", 0, 0)
			So(tunnelDir, ShouldEqual, DefaultStateDir)
			So(basePort, ShouldEqual, DefaultBasePort)
			So(maxPort, ShouldEqual, DefaultMaxPort)
		})

		Convey("When values are provided, they override the defaults", func() {
			Configure("/custom/eks-state", 10000, 10010)
			So(tunnelDir, ShouldEqual, "/custom/eks-state")
			So(basePort, ShouldEqual, 10000)
			So(maxPort, ShouldEqual, 10010)
		})

		Convey("When only some values are provided, the rest keep their current value", func() {
			tunnelDir, basePort, maxPort = DefaultStateDir, DefaultBasePort, DefaultMaxPort
			Configure("/only/dir", 0, 0)
			So(tunnelDir, ShouldEqual, "/only/dir")
			So(basePort, ShouldEqual, DefaultBasePort)
			So(maxPort, ShouldEqual, DefaultMaxPort)
		})

		Convey("When the range is inverted, it falls back to the default range", func() {
			Configure("", 9600, 9500)
			So(basePort, ShouldEqual, DefaultBasePort)
			So(maxPort, ShouldEqual, DefaultMaxPort)
		})
	})
}
