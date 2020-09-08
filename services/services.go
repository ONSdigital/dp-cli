package services

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/ONSdigital/dp-cli/out"
	"gopkg.in/yaml.v2"
)

// map [service-name][subnet]
type serviceMap map[string]map[string]service

type service struct {
	Path     string
	Priority int
	RepoURI  string
	StartCmd string
}

type Manifest struct {
	Name    string `yaml:"name"`
	RepoURI string `yaml:"repo_uri"`
	// AppType string         `yaml:"type"`
	Nomad ManifestGroups `yaml:"nomad"`
}

type ManifestGroups struct {
	Groups []ManifestClass `yaml:"groups"`
}

type ManifestClass struct {
	Class string `yaml:"class"`
	//Profiles Profile `yaml:"profiles"`
}

// Profile holds - but we are not currently interested in - cpu/mem/etc
// type Profile map[string]interface{}

var svcCache = serviceMap{}
var priorityCache = map[int]bool{}

// filter services from manifests and config
func listServices(cfg *config.Config, opts config.WithOpts) (serviceMap, error) {
	if len(svcCache) > 0 {
		return svcCache, nil
	}

	subnetsWanted := map[string]bool{"web": false, "publishing": false, "management": false}
	wantAllSubnets := !*opts.LimitPublishing && !*opts.LimitWeb
	if *opts.LimitPublishing || wantAllSubnets {
		subnetsWanted["publishing"] = true
	}
	if *opts.LimitWeb || wantAllSubnets {
		subnetsWanted["web"] = true
	}

	manDir := cfg.FindOrFromURI(filepath.Join("dp-configs", "manifests"), "")
	err := filepath.Walk(manDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			if path == manDir {
				return nil
			}
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".yml") {
			return nil
		}

		var b []byte
		if b, err = ioutil.ReadFile(path); err != nil {
			return err
		}

		manifest := Manifest{}
		if err := yaml.Unmarshal(b, &manifest); err != nil {
			return err
		}

		svcName := manifest.Name

		// build svcCache per class/subnet
		for _, grp := range manifest.Nomad.Groups {
			subnet := grp.Class

			// skip if not in subnetsWanted
			if wanted, ok := subnetsWanted[subnet]; !ok || !wanted {
				if !ok {
					fmt.Fprintf(os.Stderr, "bad class in %q for %q: %s\n", path, manifest.Name, subnet)
				}
				continue
			}

			svc := service{
				Path:    path,
				RepoURI: manifest.RepoURI,
			}

			// is svcName in config
			ignoreSubnet := false
			for svcNameCfg, svcCfg := range cfg.Services {
				if svcName != svcNameCfg {
					continue
				}
				if cfg.Services[svcName].Subnet["all"].Ignore ||
					cfg.Services[svcName].Subnet[subnet].Ignore {
					ignoreSubnet = true
					continue
				}

				// override (manifest) from config

				if len(svcCfg.Path) > 0 {
					svc.Path = svcCfg.Path
				}

				if len(svcCfg.Subnet[subnet].StartCmd) > 0 {
					svc.StartCmd = svcCfg.Subnet[subnet].StartCmd
				} else if len(svcCfg.Subnet["all"].StartCmd) > 0 {
					svc.StartCmd = svcCfg.Subnet["all"].StartCmd
				}

				if cfg.Services[svcName].Subnet["all"].Priority != 0 {
					svc.Priority = svcCfg.Subnet["all"].Priority
				} else if svcCfg.Subnet[subnet].Priority != 0 {
					svc.Priority = svcCfg.Subnet[subnet].Priority
				}

				break
			}

			if ignoreSubnet {
				continue
			}

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make(map[string]service)
			}
			svcCache[svcName][subnet] = svc
			priorityCache[svc.Priority] = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking the path %q: %v\n", manDir, err)
		return nil, err
	}

	// add any configs not in manifests, need to include "all" at this point
	subnetsWanted["all"] = true
	for svcName, svcCfg := range cfg.Services {
		for subnet := range subnetsWanted {
			if _, ok := svcCfg.Subnet[subnet]; !ok {
				continue
			}

			if !subnetsWanted[subnet] ||
				svcCfg.Subnet[subnet].Ignore ||
				svcCfg.Subnet["all"].Ignore {
				continue
			}

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make(map[string]service)
			}
			if _, ok := svcCache[svcName][subnet]; !ok {
				// we have config for a service that is not in manifests (e.g. dp-compose)
				addTo := []string{subnet}
				if subnet == "all" {
					if !svcCfg.Once {
						addTo = []string{}
						for loopWantedSubnet := range subnetsWanted {
							if loopWantedSubnet == "all" {
								continue
							}
							addTo = append(addTo, loopWantedSubnet)
						}
					}
				}
				for _, subnetAdd := range addTo {
					svcCache[svcName][subnetAdd] = service{
						Path:     svcCfg.Path,
						Priority: svcCfg.Subnet[subnet].Priority,
						RepoURI:  svcCfg.RepoURI,
						StartCmd: svcCfg.Subnet[subnet].StartCmd,
					}
					priorityCache[svcCfg.Subnet[subnet].Priority] = true
				}
			}
		}
	}

	return svcCache, nil
}

// List services
func List(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	svcs := make(serviceMap)
	if svcs, err = listServices(cfg, opts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to list services: %s\n", err)
		return err
	}

	priorities := []int{}
	for priority := range priorityCache {
		priorities = append(priorities, priority)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(priorities)))

	for priority := range priorities {
		for svcName, svc := range svcs {
			// fmt.Printf("%-40s isWeb %5v isPub %5v isMan %5v\n", svc.name, svc.IsWeb, svc.IsPublishing, svc.IsManagement)

			for subnet := range svc {
				if svc[subnet].Priority != priority {
					continue
				}

				if *opts.Verbose == 0 {
					fmt.Printf("%-40s %s\n", svcName, subnet)
				} else {
					moreArgs := []interface{}{svcName, subnet}
					moreFormats := []string{"%s"}

					if *opts.Verbose > 1 {
						moreFormats = []string{"%-12s", "%-70s", "%3d", "%s"}
						moreArgs = append(moreArgs, svc[subnet].RepoURI, svc[subnet].Priority, svc[subnet].StartCmd)
					}
					fmt.Printf("%-40s "+strings.Join(moreFormats, " ")+"\n", moreArgs...)
				}
			}
		}
	}

	return nil
}

func Clone(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	return runGit("clone", cfg, opts, args)
}

func Pull(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	return runGit("pull", cfg, opts, args)
}

func runGit(gitSub string, cfg *config.Config, opts config.WithOpts, args []string) (err error) {

	svcs := make(serviceMap)
	if svcs, err = listServices(cfg, opts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to list services: %s\n", err)
		return err
	}

	seenRepos := map[string]bool{}
	re := regexp.MustCompile("https://(github\\.com)/([^/]+)/([^/]+)")

	for svcName, svc := range svcs {
		for subnet := range svc {
			if _, ok := seenRepos[svc[subnet].RepoURI]; ok {
				if *opts.Verbose > 0 {
					fmt.Printf("Skipping seen %s in %s\n", svcName, svc[subnet].RepoURI)
				}
				continue
			}
			if matches := re.FindStringSubmatch(svc[subnet].RepoURI); len(matches) > 0 {
				githubOrg, repo := matches[1]+":"+matches[2], matches[3]
				path := filepath.Join(cfg.SourceDir, repo)

				stat, err := os.Stat(path)
				if err == nil {
					if stat.IsDir() {
						fmt.Fprintf(os.Stderr, "warning: skipping existing dir %q\n", path)
					} else {
						fmt.Fprintf(os.Stderr, "warning: skipping existing non-directory: %q\n", path)
					}
				} else if os.IsNotExist(err) {
					cmd := fmt.Sprintf("git clone git@%s/%s %s", githubOrg, repo, path)

					doCmd := true
					if *opts.Interactive {
						ch, err := out.YesOrNo("Run %s", cmd)
						if err != nil {
							return err
						}
						if ch == 'q' {
							os.Exit(1)
						} else if ch != 'y' {
							doCmd = false
						}
					}
					if doCmd {
						out.Highlight(out.INFO, "Running %s", fmt.Sprintf("git clone git@%s/%s %s", githubOrg, repo, path))
						if err = execCommand(".", "git", "clone", fmt.Sprintf("git@%s/%s", githubOrg, repo), path); err != nil {
							out.Error(err)
							os.Exit(1)
						}
						out.Highlight(out.INFO, "Done %s", repo)
					}
				} else {
					fmt.Fprintf(os.Stderr, "error: cannot stat %q: %s\n", path, err)
				}
			}
			seenRepos[svc[subnet].RepoURI] = true
		}
	}

	return nil
}

func execCommand(wrkDir, command string, arg ...string) error {
	c := exec.Command(command, arg...)
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Env = os.Environ()
	c.Dir = wrkDir
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}
