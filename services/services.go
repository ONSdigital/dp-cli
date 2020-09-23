package services

import (
	"errors"
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
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
)

// Manifest contains a service manifest file
type Manifest struct {
	Name    string         `yaml:"name"`
	RepoURI string         `yaml:"repo_uri"`
	Nomad   ManifestGroups `yaml:"nomad"`
}

// ManifestGroups has each group e.g. web, publishing
type ManifestGroups struct {
	Groups []ManifestClass `yaml:"groups"`
}

// ManifestClass has class (web, publishing, management) and tags
type ManifestClass struct {
	Class config.Subnet `yaml:"class"`
	Tags  []config.Tag  `yaml:"tags"`
	//Profiles Profile `yaml:"profiles"`
}

// Profile holds - but we are not currently interested in - cpu/mem/etc
// type Profile map[string]interface{}

// map [service-name][subnet]
type serviceMap map[string][]config.Service

const (
	listIntro = iota
	listServiceIntro
	listSubnetExtro
	listExtro
)

var svcCache = serviceMap{}
var priorityCache = map[int]bool{}
var zeroPriority = 0
var repoRegex = regexp.MustCompile("https?://([^/]+)/([^/]+)/([^/]+)")
var optsTags []config.Tag

func isIn(str string, strs []string) bool {
	for _, str1 := range strs {
		if str == str1 {
			return true
		}
	}
	return false
}

// wildcardLiteral==true requires `strs` to contain literal "*" when str=="*"
func isTagIn(tag config.Tag, tags []config.Tag, wildcardLiteral bool) bool {
	if !wildcardLiteral && tag == "*" && len(tags) > 0 {
		return true
	}
	for _, tag1 := range tags {
		if tag == tag1 {
			return true
		}
	}
	return false
}

func getTagOverlap(tagsMaybeWild []config.Tag, tags []config.Tag) (overlap []config.Tag) {
	for _, tag1 := range tagsMaybeWild {
		if isTagIn(tag1, tags, false) {
			overlap = append(overlap, tag1)
		}
	}
	return
}

// filter services from manifests and config
func listServices(cfg *config.Config, opts config.WithOpts, args []string) (serviceMap, error) {
	if len(svcCache) > 0 {
		return svcCache, nil
	}

	// convert to Tags
	for _, tag := range *opts.Tags {
		optsTags = append(optsTags, config.Tag(tag))
	}


	tagSubnetCount := map[config.Tag]map[config.Subnet]int{}

	var manDir string
	var isDir bool
	var err error
	if manDir, _, isDir, err = cfg.FindDirElseFromURI(filepath.Join("dp-configs", "manifests"), ""); err != nil {
		return nil, err
	} else if !isDir {
		return nil, errors.New("no dp-configs repo found locally")
	}
	err = filepath.Walk(manDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			out.WarnE("prevent panic by handling failure accessing a path %q: %v", path, err)
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

		// limit svcName to those on cmdline, if any
		if len(args) > 0 && !isIn(svcName, args) {
			warnAt(3, opts, "non-arg %q", svcName)
			return nil
		}

		// skip if cmdline opt ask to do so
		if opts.Skip != nil && isIn(svcName, *opts.Skip) {
			warnAt(3, opts, "non-arg %q", svcName)
			return nil
		}

		// build svcCache per class/subnet/tag from tags in manifest

		// build list of subnets=>[]tags valid for svcName
		tagsRequestedPerSubnet := map[config.Subnet][]config.Tag{}
		for _, grp := range manifest.Nomad.Groups {
			subnet := grp.Class // web, publishing, management

		ALLTAGS:
			for _, tag := range grp.Tags {

				if _, ok := tagSubnetCount[tag]; !ok {
					tagSubnetCount[tag] = make(map[config.Subnet]int)
				}
				tagSubnetCount[tag][subnet]++

				// see if cfg ignore svcName exists for this tag or '*'
				for _, svcDefault := range cfg.Services.Defaults {
					if isIn(svcName, svcDefault.Ignore) &&
						(isTagIn(tag, svcDefault.Tags, false) || isTagIn("*", svcDefault.Tags, true)) {

						warnAt(3, opts, "man ignore   %-30q in %-12q because manifest tag %-18q is in cfg default tags: %v", svcName, subnet, tag, svcDefault.Tags)
						break ALLTAGS // ignore rest of group
					}
				}

				// check if this manifest tag is in requested (cmdline) tags
				if len(optsTags) > 0 && !isTagIn(tag, optsTags, true) {
					warnAt(3, opts, "skip man     %-30q in %-12q because tag %-18q not requested", svcName, subnet, tag)
					continue
				}

				tagsRequestedPerSubnet[subnet] = append(tagsRequestedPerSubnet[subnet], tag)
				break
			}
		}

		warnAt(3, opts, "%d tags       %-30q -               valyd %+v", len(tagsRequestedPerSubnet), svcName, tagsRequestedPerSubnet)

		for subnet := range tagsRequestedPerSubnet {
			// for _, tag := range tagsRequestedPerSubnet[subnet] {
			// warnAt(3, opts, "manifest for %q in %q", svcName, subnet)

			var svcPath string
			if svcPath, _, _, err = cfg.FindDirElseFromURI(svcName, manifest.RepoURI); err != nil {
				return err
			}
			if svcPath == "" {
				if *opts.Verbose > 1 {
					out.ErrorE("no dir for %q using repo %s", svcName, manifest.RepoURI)
				}
			}

			// xref_new_svc
			svc := &config.Service{
				Path:     svcPath,
				RepoURI:  manifest.RepoURI,
				Priority: &zeroPriority,
				Subnet:   subnet,
				Tags:     tagsRequestedPerSubnet[subnet],
			}
			// if svcName == "zebedee" || svcName == "dp-dataset-api" {
			// 	warnAt(2, opts, "new\t%s\t%s\tp%3d cmd %s -- %+v", svcName, subnet, *svc.Priority, svc.StartCmd, svc)
			// }

			cfgSvcOverrides := []config.Service{}
			hasPerTagOrPerSvcMods := false

			// is cfg defaults[no-tag,tag='*' or overlap-with-wanted], then override
			for _, cfgSvcDefForTags := range cfg.Services.Defaults {
				overlapDef := getTagOverlap(tagsRequestedPerSubnet[subnet], cfgSvcDefForTags.Tags)
				if isTagIn("*", cfgSvcDefForTags.Tags, true) ||
					len(overlapDef) > 0 {

					warnAt(3, opts, "cfgOverDef   %-30q in %-12q cos cfg tags: %v", svcName, subnet, cfgSvcDefForTags.Tags)
					cfgSvcOverrides = append(cfgSvcOverrides, cfgSvcDefForTags)

					if len(overlapDef) > 0 {
						hasPerTagOrPerSvcMods = true
					}
				}
			}

			// is svcName[tag] in cfg apps, then override
			if cfgSvcs, ok := cfg.Services.Apps[svcName]; ok {
				for _, cfgSvcForTags := range cfgSvcs {
					// is svcName[tag='*',tag from svc] in cfg apps, then override
					overlap := getTagOverlap(tagsRequestedPerSubnet[subnet], cfgSvcForTags.Tags)

					// if len(cfgSvcForTags.Tags) == 0 && len(overlap) > 0 ||
					overrideType := ""
					if len(cfgSvcForTags.Tags) == 0 {
						overrideType = "Any "
					} else if isTagIn("*", cfgSvcForTags.Tags, true) {
						overrideType = "Wild"
					} else if len(overlap) > 0 {
						overrideType = "Tag "
						hasPerTagOrPerSvcMods = true
					} else {
						continue
					}
					warnAt(3, opts, "cfgOver"+overrideType+"  %-30q in %-12q cos cfg tags: %v", svcName, subnet, cfgSvcForTags.Tags)
					cfgSvcOverrides = append(cfgSvcOverrides, cfgSvcForTags)
				}
			}

			for _, cfgSvcOverride := range cfgSvcOverrides {
				svc.OverrideFrom(svcName, cfgSvcOverride)
				if svcName == "babbage" || svcName == "zdp-dataset-api" {
					warnAt(3, opts, "1a           %-30q in %-12q\tp%3d cmd %v", svcName, subnet, *svc.Priority, svc.StartCmd)
				}
			}

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make([]config.Service, 0)
			} else if !hasPerTagOrPerSvcMods {
				warnAt(2, opts, "cache---     %-30q in %12q %+v", svcName, svc.Subnet, svc)
				continue
			} else if svc.Max != nil && *svc.Max == 1 {
				warnAt(2, opts, "caMax---     %-30q in %12q %+v", svcName, svc.Subnet, svc)
				continue
			}
			svcCache[svcName] = append(svcCache[svcName], *svc)
			warnAt(3, opts, "cache+++     %-30q in %12q %+v", svcName, svc.Subnet, svc)
			priorityCache[*svc.Priority] = true

			// seenSubnets[subnet] = true
			// if svcName == "zebedee" || svcName == "dp-dataset-api" {
			// 	warnAt(3, opts, "3            %-30q in %-12q\tp%3d cmd %v", svcName, subnet, *svc.Priority, svc.StartCmd)
			// }
			// }

			// if svc.Max != nil && *svc.Max == 1 {
			// 	break // add to first found tag and no more
			// }
		}
		return nil
	})
	if err != nil {
		out.WarnE("error walking the path %q: %v", manDir, err)
		return nil, err
	}

	// warnAt(5, opts, "svcs %+v", svcCache)
	// warnAt(5, opts, "cfgSvcs %+v", cfg.Services)
	// var popularSubnet config.Subnet
	// for subnet := range seenSubnets {
	// 	popularSubnet = subnet
	// 	break
	// }

	// add any config items not already in svcCache
	for svcName, cfgSvcsByTag := range cfg.Services.Apps {
		if len(args) > 0 && !isIn(svcName, args) {
			warnAt(3, opts, "config: skipping %q arg %v", svcName, args)
			continue
		}
		// warnAt(3, opts, "ook %q", svcName)
		// first check for tag "*" (other tags later)

		// xref_new_svc
		var newSvc = &config.Service{
			Priority: &zeroPriority,
		}
		// warnAt(2, opts, "newSvc for   %-30q - %+v", svcName, newSvc)

		// is defaults[tag='*'] in cfg, then override
		for _, cfgSvcDefForTags := range cfg.Services.Defaults {
			overlapDef := getTagOverlap(optsTags, cfgSvcDefForTags.Tags)
			if len(overlapDef) != 0 || isTagIn("*", cfgSvcDefForTags.Tags, true) {
				warnAt(3, opts, "cfgOverDef   %-30q in %-12q for tags: %v", svcName, cfgSvcDefForTags.Subnet, cfgSvcDefForTags.Tags)
				newSvc.OverrideFrom(svcName, cfgSvcDefForTags)
			}
		}

		for _, cfgSvc := range cfgSvcsByTag {

			// opts asked for tags for this service, or cfgSvc has wildcard
			overlap := getTagOverlap(optsTags, cfgSvc.Tags)
			if len(overlap) == 0 && !isTagIn("*", cfgSvc.Tags, true) {
				warnAt(3, opts, "no-overlap   %-30q in %-12q has %+v want %+v", svcName, cfgSvc.Subnet, cfgSvc.Tags, optsTags)
				continue
			}
			newSvc.Tags = overlap

			// warnAt(1, opts, "all-checking %q for tag %v", svcName, wantedTag)
			// find if manifest exists (and was dealt with with "cfgSvcAll")

			// do we already have svcCache for one of these requested? if so, skip adding again

			// if wantedTag == "*" && seenAlls[svcName][nonAllTag] {
			// 	warnAt(1, opts, "seen-all %q in %q", svcName, nonAllTag)
			// 	continue
			// }

			// already exists in svcCache?
			gotSvcForSubnet := false
			for _, svcCached := range svcCache[svcName] {
				// if svcCached.Subnet == subnet {
				if len(getTagOverlap(overlap, svcCached.Tags)) > 0 {
					warnAt(3, opts, "     non-new %-30q in %-12q with tags %v", svcName, svcCached.Subnet, cfgSvc.Tags)
					gotSvcForSubnet = true
					break
				}
				// }
			}
			if gotSvcForSubnet {
				warnAt(3, opts, "cfgRepSkip   %-30q in %-12q has %+v", svcName, cfgSvc.Subnet, cfgSvc.Tags)
				continue
			}

			// override from this config
			newSvc.OverrideFrom(svcName, cfgSvc)

			if newSvc.Path == "" {
				if newSvc.Path, _, _, err = cfg.FindDirElseFromURI(svcName, ""); err != nil {
					return nil, err
				}
				warnAt(2, opts, "found       %-30q a path %s", newSvc.Path)
			}

			var popularSubnet config.Subnet
			var subnetScore = -1
			for tagForSubnet := range tagSubnetCount {
				for subnetForTag := range tagSubnetCount[tagForSubnet] {
					if tagSubnetCount[tagForSubnet][subnetForTag] > subnetScore {
						subnetScore = tagSubnetCount[tagForSubnet][subnetForTag]
						popularSubnet = subnetForTag
					}
				}
			}
			newSvc.Subnet = popularSubnet

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make([]config.Service, 0)
			} else if newSvc.Max != nil && *newSvc.Max == 1 {
				warnAt(3, opts, "cfgMax--     %-30q in %-12q with tags %v", svcName, newSvc.Subnet, cfgSvc.Tags)
				break // add to first found tag and no more
			}

			warnAt(3, opts, "cfg+++++     %-30q in %-12q with tags %v", svcName, newSvc.Subnet, cfgSvc.Tags)
			svcCache[svcName] = append(svcCache[svcName], *newSvc)
			priorityCache[*newSvc.Priority] = true

		}
	}
	warnAt(3, opts, "cache from manifest %+v", svcCache["zebedee"])
	return svcCache, nil
}

// List services
func List(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	svcs := make(serviceMap)
	if svcs, err = listServices(cfg, opts, args); err != nil {
		out.WarnE("failed to list services: %s", err)
		return err
	}

	if *opts.Verbose >= 4 {
		spew.Dump(svcs)
	}

	if *opts.Itermaton {
		itermaton(listIntro, "", "", nil, opts, nil, nil)
	}
	counts := map[string]int{"wins": 0, "tabs": 0, "tabs_in_win": 0, "panes": 0, "panes_in_tab": 0}

	// collect svcName(s) by tag, so we can sort within that tag
	svcNamesOrderBySubnetTag := map[config.Subnet]map[string][]string{}
	subnets := []string{}
	for svcName, svcsByTag := range svcs {
		// fmt.Printf("%-40s isWeb %5v isPub %5v isMan %5v\n", svc.name, svc.IsWeb, svc.IsPublishing, svc.IsManagement)

		for _, svcByTag := range svcsByTag {
			var tagList []string
			doAppend := false
			if len(svcByTag.Tags) == 0 {
				warnAt(3, opts, "noTag        %-30q", svcName)
				doAppend = true
				tagList = append(tagList, "*")
			} else {
				for _, tag := range svcByTag.Tags {
					if len(optsTags) == 0 ||
						tag == "*" || isTagIn(tag, optsTags, true) {

						doAppend = true
						tagList = append(tagList, string(tag))
					}
				}
			}
			if doAppend {
				sort.Strings(tagList)
				tagListStr := strings.Join(tagList, ",")
				if _, ok := svcNamesOrderBySubnetTag[svcByTag.Subnet]; !ok {
					svcNamesOrderBySubnetTag[svcByTag.Subnet] = make(map[string][]string)
				}
				svcNamesOrderBySubnetTag[svcByTag.Subnet][tagListStr] = append(svcNamesOrderBySubnetTag[svcByTag.Subnet][tagListStr], svcName)
				if !isIn(string(svcByTag.Subnet), subnets) {
					subnets = append(subnets, string(svcByTag.Subnet))
				}
			}
		}
	}
	warnAt(4, opts, "ordered tags %+v", svcNamesOrderBySubnetTag)
	sort.Strings(subnets)

	for _, subnetStr := range subnets {
		subnet := config.Subnet(subnetStr)

		for tagListStr, svcNames := range svcNamesOrderBySubnetTag[subnet] {
			// for tag, svcNames := range svcs {
			sort.Strings(svcNames)

			for _, svcName := range svcNames {
				// for svcName := range svcs {
				for _, svc := range svcs[svcName] {
					// if !isTagIn(tag, svc.Tags, false) {
					// 	continue
					// }
					if svc.Subnet != subnet {
						continue
					}

					warnAt(3, opts, "got svcName  %-30q for tags %v for list %q", svcName, svc.Tags, tagListStr)

					if *opts.Itermaton {
						itermaton(listServiceIntro, svcName, tagListStr, &svc, opts, counts, &subnet)

					} else if *opts.Verbose == 0 {
						fmt.Printf("%-40s %s\n", svcName, tagListStr)

					} else {
						moreArgs := []interface{}{svcName, tagListStr, *svc.Priority}
						moreFormats := []string{"%-40s %-20s", "%3d"}

						if *opts.Verbose > 1 {
							moreFormats = append(moreFormats, "%-60s %v %v")
							moreArgs = append(moreArgs, config.ShellifyPath(svc.Path), svc.StartCmd, svc.StopCmd)
							if *opts.Verbose > 2 {
								moreFormats = append(moreFormats, "%v %v")
								moreArgs = append(moreArgs, svc.InitCmds, svc.LateInitCmds)
							}
						}
						fmt.Printf(strings.Join(moreFormats, " ")+"\n", moreArgs...)
					}
					break
				}
			}
		}
		if *opts.Itermaton {
			itermaton(listSubnetExtro, "", "", nil, opts, counts, &subnet)
		}
	}

	if *opts.Itermaton {
		itermaton(listExtro, "", "", nil, opts, counts, nil)
	}

	return nil
}

// Start can start the services given
func Start(cfg *config.Config, opts config.WithOpts, args []string) (err error) {

	var svcs serviceMap
	if svcs, err = listServices(cfg, opts, args); err != nil {
		out.WarnE("failed to list services: %s", err)
		return err
	}

	for svcName, svcsByTag := range svcs {
		for subnet, svc := range svcsByTag {
			if len(svc.StartCmd) == 0 {
				svc.StartCmd = []string{"make", "debug"}
			}
			doCmd := true
			cmd := []string{} // XXX TODO
			cmd = append(cmd, svc.StartCmd...)

			if *opts.Interactive {
				ch, err := out.YesOrNo("Run %s", strings.Join(cmd, " "))
				if err != nil {
					return err
				}
				if ch == 'q' {
					os.Exit(0)
				} else if ch != 'y' {
					doCmd = false
				}
			}
			if doCmd {
				if err = execCommandWrap(svcName, opts, svc.Path, cmd...); err != nil {
					out.WarnE("exec for %q in %q failed: %s", svcName, subnet, err)
				}
			}
		}
	}
	return
}

// Clone can clone the services given
func Clone(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	return execOrClone(true, cfg, opts, args)
}

// Exec runs a command in the repo dir for services
func Exec(cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	return execOrClone(false, cfg, opts, args)
}

func execOrClone(isClone bool, cfg *config.Config, opts config.WithOpts, args []string) (err error) {
	svcs := make(serviceMap)
	if svcs, err = listServices(cfg, opts, args); err != nil {
		out.WarnE("failed to list services: %s", err)
		return err
	}

	seenRepos := map[string]bool{}

	for svcName, svc := range svcs {
		for subnet := range svc {
			if _, ok := seenRepos[svc[subnet].RepoURI]; ok {
				warnAt(2, opts, "Skipping seen %s in %s\n", svcName, svc[subnet].RepoURI)
				continue
			}

			// XXX isClone needs parent (and bail if is sub-dir)
			// !isClone needs repo dir (and check for dir)
			path := svc[subnet].Path
			var githubOrg, repo, cwd string
			var cmd []string
			var isExist, isDir bool

			// get path from RepoURI, get githubOrg
			if matches := repoRegex.FindStringSubmatch(svc[subnet].RepoURI); len(matches) > 0 {
				githubOrg, repo = matches[1]+":"+matches[2], matches[3]
				if path, isExist, isDir, err = cfg.FindDirElseFromURI(repo, svc[subnet].RepoURI); err != nil {
					return err
				}
			} else {
				// check path exists
				if isExist, isDir, err = config.IsExistDir(path); err != nil {
					out.WarnE("error: cannot stat %q: %s", path, err)
					return
				}
			}
			if isExist {
				if !isDir {
					out.WarnE("warning: skipping existing (but non-directory): %q", path)
					continue
				}
				if isClone {
					out.WarnE("warning: skipping existing target dir %q", path)
					continue
				}
				// !isClone, isDir (isExist)
				cmd = args
				cwd = path
			} else {
				if !isClone {
					out.WarnE("warning: skipping non-existing directory: %q", path)
					continue
				}
				// isClone and !isExist
				cwd = filepath.Dir(path)
				if _, parentIsDir, err := config.IsExistDir(cwd); err != nil || !parentIsDir {
					if err != nil {
						out.ErrorE("stat on repo parent: %q %s", cwd, err)
						return err
					}
					return fmt.Errorf("non-existing repo parent: %q", cwd)
				}
				cmd = []string{"git", "clone", fmt.Sprintf("git@%s/%s", githubOrg, repo), path}
			}

			if err = execCommandWrap(svcName, opts, cwd, cmd...); err != nil {
				return
			}
			seenRepos[svc[subnet].RepoURI] = true
		}
	}
	return nil
}

func execCommandWrap(svcName string, opts config.WithOpts, cwd string, cmd ...string) (err error) {
	cmdStr := strings.Join(cmd, " ")
	if *opts.Interactive {
		var ch byte
		if ch, err = out.YesOrNo("Run %s in dir %s", cmdStr, config.ShellifyPath(cwd)); err != nil {
			return
		}
		if ch == 'q' {
			os.Exit(0)
		} else if ch != 'y' {
			return
		}
	}

	if *opts.Verbose > 1 {
		out.InfoE("For %q - running %q in %q", svcName, cmdStr, cwd)
	}
	if err = execCommand(cwd, cmd[0], cmd[1:]...); err != nil {
		return
	}
	if *opts.Verbose > 0 {
		out.InfoE("Done %q in %q", svcName, cwd)
	}
	return
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

func warnAt(minLevel int, opts config.WithOpts, fmt string, args ...interface{}) {
	if *opts.Verbose >= minLevel {
		out.WarnE(fmt, args...)
	}
}

func jsonSafe(str string, enQuote bool) string {
	q := ``
	if enQuote {
		q = `\"`
	}
	return q + strings.ReplaceAll(str, `"`, `\"`) + q
}

func itermaton(do int, svcName, tagListStr string, svc *config.Service, opts config.WithOpts, counts map[string]int, subnet *config.Subnet) {
	panesPerTab := 8
	if opts.PanesPerTag != nil {
		panesPerTab = *opts.PanesPerTag
	}

	if do == listIntro {
		fmt.Println(`{
	"profile":        "itermaton",
	"start_command":  "make debug",
	"init_commands":  ["export DP_CLI_APP_VERSION=` + jsonSafe(opts.AppVersion, true) + `"],
	"late_init_commands":  ["cd ` + jsonSafe("$DP_CLI_SVC_PATH", true) + `"],
	"stop_is_sigint": true,
	"debug":          false,
	"windows":        [`,
		)

	} else if do == listServiceIntro {
		if counts["tabs_in_win"] == 0 {
			if counts["wins"] > 0 {
				fmt.Println(",") // win separator
			}
			fmt.Printf(`  {"name":"%s win-%d", "tabs":[`+"\n", *subnet, counts["wins"]) // new window
			counts["wins"]++
			counts["panes_in_tab"] = 0
		}

		if counts["panes_in_tab"] == 0 {
			if counts["tabs_in_win"] > 0 {
				fmt.Println(",") // tab separator
			}
			fmt.Printf(`    {"name":"%s tab-%d %s", "panes":[`+"\n", tagListStr, counts["tabs"], svcName) // new tab
			counts["tabs"]++
			counts["tabs_in_win"]++
		}

		if counts["panes_in_tab"] > 0 {
			fmt.Println(",") // pane separator
		}

		// build pane
		extra := ""
		if counts["panes_in_tab"] == panesPerTab/2 {
			extra = `,"startsNextRow":true`
		}

		if svc.Priority != nil && *svc.Priority != 0 {
			extra += fmt.Sprintf(`,"priority":%d`, *svc.Priority)
		}

		extra += `,"init_commands":[` +
			strings.Join([]string{
				`"export DP_CLI_SVC_NAME=` + jsonSafe(svcName, true) + `"`,
				`"export DP_CLI_SVC_PATH=` + jsonSafe(svc.Path, true) + `"`,
			}, ",")
		for _, initCmd := range svc.InitCmds {
			extra += `,"` + jsonSafe(initCmd, false) + `"`
		}
		extra += `]`

		if len(svc.LateInitCmds) > 0 {
			extra += `,"late_init_commands":[`
			comma := ""
			for _, initCmd := range svc.LateInitCmds {
				extra += comma + `"` + jsonSafe(initCmd, false) + `"`
				comma = ","
			}
			extra += `]`
		}

		if len(svc.StartCmd) > 0 {
			extra += `,"start_command":"` + jsonSafe(strings.Join(svc.StartCmd, " "), false) + `"`
		}
		if len(svc.StopCmd) > 0 {
			extra += `,"stop_command":"` + jsonSafe(strings.Join(svc.StopCmd, " "), false) + `"`
		}
		fmt.Printf(`      {"name":"%s"%s}`, svcName, extra) // pane
		counts["panes"]++
		counts["panes_in_tab"]++

		if counts["panes_in_tab"] == panesPerTab {
			fmt.Print("\n    ]}") // end panes
			counts["panes_in_tab"] = 0
		}
	} else if do == listSubnetExtro {

		if counts["panes_in_tab"] > 0 {
			fmt.Print("\n    ]}") // end panes
		}
		// counts["tabs"]++
		fmt.Print("\n  ]}") // end tabs
		counts["tabs_in_win"] = 0

	} else if do == listExtro {
		fmt.Println("\n]}") // end windows

	}
}
