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
	"strconv"
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

// serviceMap[service-name] -> []Service
type serviceMap map[string][]config.Service

const (
	listIntro = iota
	listServiceIntro
	listServiceExtro
	listSubnetExtro
	listExtro
)

var (
	svcCache               = serviceMap{}
	tagsRequestedPerSubnet = map[config.Subnet]map[string][]config.Tag{}
	zeroPriority           = 0
	repoRegex              = regexp.MustCompile("https?://([^/]+)/([^/]+)/([^/]+)")
	optsTags               []config.Tag
)

func isIn(str string, strs []string) bool {
	for _, str1 := range strs {
		if str == str1 {
			return true
		}
	}
	return false
}

// tagWildcardLiteral==true requires `tags` to contain literal "*" when tag=="*"
func isTagIn(tag config.Tag, tags []config.Tag, tagWildcardLiteral bool) bool {
	if !tagWildcardLiteral && tag == "*" && len(tags) > 0 {
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

// filter services from manifests, config and opts
func listServices(cfg *config.Config, opts config.WithOpts, args []string) (serviceMap, error) {
	if len(svcCache) > 0 {
		return svcCache, nil
	}

	// convert Tags to Tag type
	for _, tag := range *opts.Tags {
		optsTags = append(optsTags, config.Tag(tag))
	}

	tagSubnetCount := map[config.Tag]map[config.Subnet]int{}

	var manifestDir string
	var isDir bool
	var err error
	if manifestDir, _, isDir, err = cfg.FindDirElseFromURI(filepath.Join("dp-configs", "manifests"), ""); err != nil {
		return nil, err
	} else if !isDir {
		return nil, errors.New("no dp-configs repo found locally")
	}
	err = filepath.Walk(manifestDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			out.WarnE("walk error with path %q: %v", path, err)
			return err
		}
		if info.IsDir() {
			if path == manifestDir {
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

		// limit svcName to those opted for on cmdline, if any
		if len(*opts.Svcs) > 0 && !isIn(svcName, *opts.Svcs) {
			opts.WarnAtSvc(3, "non-opt", svcName, "")
			return nil
		}

		// skip cmdline opts
		if opts.Skips != nil && isIn(svcName, *opts.Skips) {
			opts.WarnAtSvc(3, "skip-arg", svcName, "")
			return nil
		}

		// build svcCache per class/subnet/tag from tags in manifest

		// build list of subnets=>[]tags valid for svcName
		for _, grp := range manifest.Nomad.Groups {
			subnet := grp.Class // web, publishing, management

		ALLTAGS:
			for _, tag := range grp.Tags {

				if _, ok := tagSubnetCount[tag]; !ok {
					tagSubnetCount[tag] = make(map[config.Subnet]int)
				}
				tagSubnetCount[tag][subnet]++

				// see if cfg defaults.ignore[svcName] exists for this tag or '*'
				for _, svcDefault := range cfg.Services.Defaults {
					if isIn(svcName, svcDefault.Ignore) &&
						(isTagIn(tag, svcDefault.Tags, false) || isTagIn("*", svcDefault.Tags, true)) {

						opts.WarnAtSvc(3, "man ignore", svcName, "in %-12q because manifest tag %-18q is in cfg default tags: %v", subnet, tag, svcDefault.Tags)
						break ALLTAGS // ignore rest of group(subnet)
					}
				}

				// skip unless this manifest tag is in requested cmdline tags
				if len(optsTags) > 0 && !isTagIn(tag, optsTags, true) {
					opts.WarnAtSvc(3, "skip man", svcName, "in %-12q because tag %-18q not requested", subnet, tag)
					continue
				}

				if _, ok := tagsRequestedPerSubnet[subnet]; !ok {
					tagsRequestedPerSubnet[subnet] = make(map[string][]config.Tag)
				}
				tagsRequestedPerSubnet[subnet][svcName] = append(tagsRequestedPerSubnet[subnet][svcName], tag)
				// break
			}
		}

		opts.WarnAtSvc(3, strconv.Itoa(len(tagsRequestedPerSubnet))+" tags", svcName, "-               valyd %+v", tagsRequestedPerSubnet)

		for subnet := range tagsRequestedPerSubnet {

			if _, ok := tagsRequestedPerSubnet[subnet][svcName]; !ok {
				opts.WarnAtSvc(3, "cfgNoSubnt", svcName, "in %-12q SKIP", subnet)
				continue
			}

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
				Tags:     tagsRequestedPerSubnet[subnet][svcName],
			}

			cfgSvcOverrides := []config.Service{}
			hasPerTagOrPerSvcMods := false

			// is cfg defaults[no-tag,tag='*' or overlap-with-wanted], then override
			for _, cfgSvcDefForTags := range cfg.Services.Defaults {
				overlapDef := getTagOverlap(tagsRequestedPerSubnet[subnet][svcName], cfgSvcDefForTags.Tags)
				if isTagIn("*", cfgSvcDefForTags.Tags, true) || len(overlapDef) > 0 {

					opts.WarnAtSvc(3, "cfgOverDef", svcName, "in %-12q cos cfg tags: %v", subnet, cfgSvcDefForTags.Tags)
					cfgSvcOverrides = append(cfgSvcOverrides, cfgSvcDefForTags)

					if len(overlapDef) > 0 {
						hasPerTagOrPerSvcMods = true
					}
				}
			}

			// is svcName[tag] in cfg apps, then override
			if cfgSvcs, ok := cfg.Services.Apps[svcName]; ok {
				for _, cfgSvcForTags := range cfgSvcs {
					if cfgSvcForTags.Subnet != "" && cfgSvcForTags.Subnet != subnet {
						opts.WarnAtSvc(3, "noOverSubnet", svcName, "in %-12q cos cfg subnet: %v", subnet, cfgSvcForTags.Subnet)
						continue
					}

					// is svcName[tag='*',tag from svc] in cfg apps, then override
					overlap := getTagOverlap(tagsRequestedPerSubnet[subnet][svcName], cfgSvcForTags.Tags)
					overrideType := ""
					if len(cfgSvcForTags.Tags) == 0 {
						overrideType = "cfgOverAny"
					} else if isTagIn("*", cfgSvcForTags.Tags, true) {
						overrideType = "cfgOverWild"
					} else if len(overlap) > 0 {
						overrideType = "cfgOverTag"
						hasPerTagOrPerSvcMods = true
					} else {
						continue
					}
					opts.WarnAtSvc(3, overrideType, svcName, "in %-12q cos cfg tags: %v", subnet, cfgSvcForTags.Tags)
					cfgSvcOverrides = append(cfgSvcOverrides, cfgSvcForTags)
				}
			}

			for _, cfgSvcOverride := range cfgSvcOverrides {
				svc.OverrideFrom(svcName, cfgSvcOverride)
				opts.WarnAtSvc(4, "1a", svcName, "in %-12q\tp%3d cmd %v", subnet, *svc.Priority, svc.StartCmd)
			}

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make([]config.Service, 0)
			} else if !hasPerTagOrPerSvcMods {
				opts.WarnAtSvc(2, "cache---", svcName, "in %12q %+v", svc.Subnet, svc)
				continue
			} else if svc.Max != nil && *svc.Max == 1 {
				opts.WarnAtSvc(2, "caMax---", svcName, "in %12q %+v", svc.Subnet, svc)
				continue
			}
			svcCache[svcName] = append(svcCache[svcName], *svc)
			opts.WarnAtSvc(3, "cache+++", svcName, "in %12q %d %+v", svc.Subnet, len(svcCache[svcName]), svc)

		}
		return nil
	})
	if err != nil {
		out.WarnE("error walking the path %q: %v", manifestDir, err)
		return nil, err
	}

	opts.WarnAt(5, "svcs %+v", svcCache)
	opts.WarnAt(5, "cfgSvcs %+v", cfg.Services)

	// add any config items not already in svcCache
	for svcName, cfgSvcsByTag := range cfg.Services.Apps {
		if len(*opts.Svcs) > 0 && !isIn(svcName, *opts.Svcs) {
			opts.WarnAtSvc(3, "config", svcName, "skipping - not in svcs %v", *opts.Svcs)
			continue
		}
		if len(*opts.Skips) > 0 && isIn(svcName, *opts.Skips) {
			opts.WarnAtSvc(3, "config", svcName, "skipping arg %v", *opts.Skips)
			continue
		}

		// first check for tag "*" in defaults (other tags later)
		// if defaults[tag='*'] exists in cfg, then use that as override first (more specific later)
		defaultOverrides := []*config.Service{}
		for _, cfgSvcDefForTags := range cfg.Services.Defaults {
			overlapDef := getTagOverlap(optsTags, cfgSvcDefForTags.Tags)
			if len(overlapDef) != 0 || isTagIn("*", cfgSvcDefForTags.Tags, true) {
				opts.WarnAtSvc(3, "cfgOverDEF", svcName, "in %-12q for tags: %v", cfgSvcDefForTags.Subnet, cfgSvcDefForTags.Tags)
				defaultOverrides = append(defaultOverrides, &cfgSvcDefForTags)
			}
		}

		for _, cfgSvc := range cfgSvcsByTag {

			if len(cfgSvc.Tags) == 0 {
				opts.WarnAtSvc(3, "cfgNoTags", svcName, "in %-12q", cfgSvc.Subnet)
				continue
			}

			// xref_new_svc
			var newSvc = &config.Service{
				Priority: &zeroPriority,
			}
			for _, defaultOverride := range defaultOverrides {
				newSvc.OverrideFrom(svcName, *defaultOverride)
			}

			// opts asked for tags for this service, or cfgSvc has wildcard
			var overlap []config.Tag
			if isTagIn("*", cfgSvc.Tags, true) || // want cfgSvc if has tag "*"
				(len(optsTags) == 0 && len(cfgSvc.Tags) > 0) { // want cfgSvc if no optsTags and cfgSvc has tags

				overlap = cfgSvc.Tags

			} else if len(optsTags) > 0 {

				// both opts and cfgSvc have tags
				overlap = getTagOverlap(optsTags, cfgSvc.Tags)
				if len(overlap) == 0 {
					opts.WarnAtSvc(3, "not-overlap", svcName, "in %-12q has %+v want %+v", cfgSvc.Subnet, cfgSvc.Tags, optsTags)
					continue
				}
			}

			// cfgSvc already exists in svcCache?
			seenSvcForSubnet := false
			for _, svcCached := range svcCache[svcName] {
				if svcCached.Subnet == cfgSvc.Subnet ||
					len(getTagOverlap(overlap, svcCached.Tags)) > 0 {

					opts.WarnAtSvc(3, "non-new", svcName, "in %-12q with tags %v", svcCached.Subnet, cfgSvc.Tags)
					seenSvcForSubnet = true
					break
				}
			}
			if seenSvcForSubnet {
				opts.WarnAtSvc(3, "cfgRepSkip", svcName, "in %-12q has %+v", cfgSvc.Subnet, cfgSvc.Tags)
				continue
			}

			newSvc.Tags = overlap

			// override from this config
			newSvc.OverrideFrom(svcName, cfgSvc)

			if newSvc.Path == "" {
				if newSvc.Path, _, _, err = cfg.FindDirElseFromURI(svcName, ""); err != nil {
					return nil, err
				}
				opts.WarnAtSvc(3, "found", svcName, "a path %s", newSvc.Path)
			}

			// find the right subnet for svcName tags
			if cfgSvc.Subnet != "" {
				newSvc.Subnet = cfgSvc.Subnet
			} else {
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
				opts.WarnAtSvc(3, "cfgSubnet!", svcName, "in %-12q with tags %v", newSvc.Subnet, cfgSvc.Tags)
			}

			if _, ok := svcCache[svcName]; !ok {
				svcCache[svcName] = make([]config.Service, 0)
			} else if newSvc.Max != nil && *newSvc.Max == 1 {
				opts.WarnAtSvc(3, "cfgMax--", svcName, "in %-12q with tags %v", newSvc.Subnet, cfgSvc.Tags)
				break // add to first found tag and no more
			}

			svcCache[svcName] = append(svcCache[svcName], *newSvc)
			opts.WarnAtSvc(3, "cfg+++++", svcName, "in %-12q %d with tags %v", newSvc.Subnet, len(svcCache[svcName]), cfgSvc.Tags)

			if _, ok := tagsRequestedPerSubnet[newSvc.Subnet]; !ok {
				tagsRequestedPerSubnet[newSvc.Subnet] = make(map[string][]config.Tag, 0)
			}
			tagsRequestedPerSubnet[newSvc.Subnet][svcName] = newSvc.Tags

		}
	}
	opts.WarnAt(3, "cache from manifest %+v", svcCache["zebedee"])
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

	itermaton(listIntro, "", cfg, "", nil, opts, nil, nil)

	counts := map[string]int{"wins": 0, "tabs": 0, "tabs_in_win": 0, "panes": 0, "panes_in_tab": 0}

	for subnet := range tagsRequestedPerSubnet {
		svcNames := make([]string, len(tagsRequestedPerSubnet[subnet]))
		for svcName := range tagsRequestedPerSubnet[subnet] {
			svcNames = append(svcNames, svcName)
		}
		sort.Strings(svcNames)

		for _, svcName := range svcNames {
			tagListStrings := []string{}
			for _, tag := range tagsRequestedPerSubnet[subnet][svcName] {
				tagListStrings = append(tagListStrings, string(tag))
			}
			tagListStr := string(subnet) + "[" + strings.Join(tagListStrings, ",") + "]"

			for _, svc := range svcs[svcName] {
				if svc.Subnet != subnet {
					continue
				}

				opts.WarnAtSvc(3, "got svcName", svcName, "for tags %v for list %q", svc.Tags, tagListStr)

				if *opts.Itermaton {
					itermaton(listServiceIntro, svcName, cfg, tagListStr, &svc, opts, counts, &subnet)
					itermaton(listServiceExtro, svcName, cfg, tagListStr, &svc, opts, counts, &subnet)

				} else if *opts.Verbose == 0 {
					fmt.Printf("%-40s %s\n", svcName, tagListStr)

				} else {
					moreArgs := []interface{}{svcName, tagListStr, *svc.Priority}
					moreFormats := []string{"%-40s %-20s", "%3d"}

					if *opts.Verbose > 1 {
						moreFormats = append(moreFormats, "%-60s %v %v")
						moreArgs = append(moreArgs, config.ShellifyPath(svc.Path), svc.StartCmd, svc.StopCmd)
						if *opts.Verbose > 4 {
							moreFormats = append(moreFormats, "%v %v")
							moreArgs = append(moreArgs, svc.InitCmds, svc.LateInitCmds)
						}
					}
					fmt.Printf(strings.Join(moreFormats, " ")+"\n", moreArgs...)
				}

				break // done service for this subnet
			}
		}

		itermaton(listSubnetExtro, "", cfg, "", nil, opts, counts, &subnet)

	}

	if *opts.Itermaton {
		itermaton(listExtro, "", cfg, "", nil, opts, counts, nil)
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
				opts.WarnAtSvc(3, "Skipping", svcName, "in %s - seen", svc[subnet].RepoURI)
				continue
			}
			seenRepos[svc[subnet].RepoURI] = true

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
					return
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

			out.InfoFHighlight("In %s\t\t%s", config.ShellifyPath(path), svc[subnet].RepoURI)
			if err = execCommandWrap(svcName, opts, cwd, cmd...); err != nil {
				return
			}
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

func jsonSafe(str string, enQuote bool) string {
	q := ``
	if enQuote {
		q = `\"`
	}
	return q + strings.ReplaceAll(str, `"`, `\"`) + q
}

func itermaton(do int, svcName string, cfg *config.Config, tagListStr string, svc *config.Service, opts config.WithOpts, counts map[string]int, subnet *config.Subnet) {
	if !*opts.Itermaton {
		return
	}
	shapeInts := []int{3, 4, 2}
	shape := config.Itermatons{MaxTabs: &shapeInts[0], MaxColumns: &shapeInts[1], MaxRows: &shapeInts[2]}
	if cfg.Itermaton.MaxTabs != nil {
		shape.MaxTabs = cfg.Itermaton.MaxTabs
	}
	if cfg.Itermaton.MaxColumns != nil {
		shape.MaxColumns = cfg.Itermaton.MaxColumns
	}
	if cfg.Itermaton.MaxRows != nil {
		shape.MaxRows = cfg.Itermaton.MaxRows
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
		if counts["panes_in_tab"] == *shape.MaxColumns**shape.MaxRows {
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

		if counts["panes_in_tab"] == *shape.MaxColumns**shape.MaxRows {
			fmt.Print("\n    ]}") // end panes
			counts["panes_in_tab"] = 0
		}

	} else if (do == listSubnetExtro && cfg.Itermaton.WinPerSubnet != nil && *cfg.Itermaton.WinPerSubnet) ||
		(do == listServiceExtro && cfg.Itermaton.MaxTabs != nil && counts["tabs_in_win"] >= *cfg.Itermaton.MaxTabs) ||
		do == listExtro {

		if counts["panes_in_tab"] > 0 {
			fmt.Print("\n    ]}") // end panes
		}
		fmt.Print("\n  ]}") // end tabs
		counts["tabs_in_win"] = 0

		if do == listExtro {
			fmt.Println("\n]}") // end windows
		}
	}
}
