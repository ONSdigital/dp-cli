package config

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/out"
	"gopkg.in/yaml.v2"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

// Config holds the config file contents
type Config struct {
	CMD          CMD           `yaml:"cmd"`
	Environments []Environment `yaml:"environments"`
	SSHUser      string        `yaml:"ssh-user"`
	SourcePath   []string      `yaml:"dp-source-path"`
	Services     ServiceWrap   `yaml:"services"`
	Itermaton    Itermatons    `yaml:"itermaton"`
}

// CMD has some data related info
type CMD struct {
	MongoURL    string   `yaml:"mongo-url"`
	Neo4jURL    string   `yaml:"neo4j-url"`
	MongoDBs    []string `yaml:"mongo-dbs"`
	Hierarchies []string `yaml:"hierarchies"`
	Codelists   []string `yaml:"codelists"`
}

// Environment represents an environment
type Environment struct {
	Name       string     `yaml:"name"`
	Profile    string     `yaml:"profile"`
	ExtraPorts ExtraPorts `yaml:"extra-ports"`
}

// ExtraPorts is a list of ports for the given Security Group
type ExtraPorts struct {
	Bastion    []int64 `yaml:"bastion"`
	Publishing []int64 `yaml:"publishing"`
	Web        []int64 `yaml:"web"`
}

// Tag is really a journey tag
type Tag string

// Subnet should be web,publishing (maybe management)
type Subnet string

// ServiceWrap is the services section of config
type ServiceWrap struct {
	Defaults []Service            `yaml:"defaults"`
	Apps     map[string][]Service `yaml:"apps"`
}

// Service allows individual configuration of a service for the given tags
type Service struct {
	Tags         []Tag    `yaml:"tags"`
	Path         string   `yaml:"path"`
	RepoURI      string   `yaml:"repo_uri"`
	Count        *int     `yaml:"count"`
	Max          *int     `yaml:"max"`
	Priority     *int     `yaml:"priority"`
	InitCmds     []string `yaml:"init_commands"`
	LateInitCmds []string `yaml:"late_init_commands"`
	StartCmd     []string `yaml:"start_command"`
	StopCmd      []string `yaml:"stop_command"`
	Ignore       []string `yaml:"ignore"`
	Subnet       Subnet
}

// Itermatons are config options for itermatons output
type Itermatons struct {
	MaxTabs      *int  `yaml:"max_tabs"`
	MaxColumns   *int  `yaml:"max_columns"`
	MaxRows      *int  `yaml:"max_rows"`
	WinPerSubnet *bool `yaml:"window_per_subnet"`
}

// WithOpts is used to pass cmdline opts to commands
type WithOpts struct {
	AppVersion  string
	ForUser     *string
	HTTPOnly    *bool
	Interactive *bool
	Itermaton   *bool
	Svcs        *[]string
	Skips       *[]string
	Tags        *[]string
	Verbose     *int
}

// Get returns the config struct by parsing the YML file
func Get() (*Config, error) {
	path := os.Getenv("DP_CLI_CONFIG")
	if len(path) == 0 {
		var err error
		path, err = getDefaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.SourcePath) == 0 {
		return nil, errors.New("need dp-source-path in config")
	}
	// convert ~ to home dir in paths, or prepend SourceDir if relative
	for idx := range cfg.SourcePath {
		if cfg.SourcePath[idx], _, _, err = cfg.expandPath(cfg.SourcePath[idx]); err != nil {
			return nil, err
		}
	}
	for svcName, svcs := range cfg.Services.Apps {
		for sIdx, svc := range svcs {
			for _, path := range []string{svc.Path, svcName} {
				if path != "" {
					var newPath string
					if newPath, _, _, err = cfg.FindDirElseFromURI(path, svc.RepoURI); err != nil {
						return nil, err
					}
					cfg.Services.Apps[svcName][sIdx].Path = newPath
					if cfg.Services.Apps[svcName][sIdx].Path != "" {
						break
					}
				}
			}
		}
	}
	return &cfg, nil
}

// IsExistDir returns existance and isDir for `dir`
func IsExistDir(dir string) (exists, isDir bool, err error) {
	var s os.FileInfo
	if s, err = os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
	} else {
		exists = true
		isDir = s.IsDir()
	}
	return
}

// FindDirInPath returns empty string if no dir `dir` exists in `path` (and isExist is true when `dir` exists in `path`)
func FindDirInPath(dir string, path []string) (fullPath string, isExist bool, err error) {
	for _, d := range path {
		var isDir bool
		fullPath = filepath.Join(d, dir)
		if isExist, isDir, err = IsExistDir(fullPath); err != nil {
			return // err
		} else if isDir {
			return // fullPath, true
		} else if isExist {
			return "", isExist, nil // "",true => non-dir exists!
		}
	}
	return "", false, nil
}

// FindDirElseFromURI expands path or (if path empty) URI converted into full path
func (cfg *Config) FindDirElseFromURI(path, uri string) (string, bool, bool, error) {
	if path == "" {
		if uri != "" {
			lastSlashAt := strings.LastIndex(uri, "/")
			if lastSlashAt == -1 {
				return "", false, false, fmt.Errorf("Bad uri %s", uri)
			}
			path = uri[lastSlashAt:]
		}
	}
	return cfg.expandPath(path)
}

// expandPath
// - return "" if path is ""
// - expands leading ~
// - returns `path` if starts with / (no check for existance)
// - returns `SourcePath[N]/path` when that exists and is a dir
// - returns `SourcePath[0]/path` (no check)
func (cfg *Config) expandPath(path string) (newPath string, isExist, isDir bool, err error) {
	if path == "" {
		return "", false, false, nil
	}
	if strings.HasPrefix(path, "~/") {
		path = os.Getenv("HOME") + path[1:]
	}
	if strings.HasPrefix(path, "/") {
		return path, false, false, nil
	}
	if newPath, isExist, err = FindDirInPath(path, cfg.SourcePath); err != nil {
		return "", false, false, err
	} else if newPath != "" {
		return newPath, true, true, nil
	}
	return filepath.Join(cfg.SourcePath[0], path), false, false, nil
}

// ShellifyPath converts home dir to ~ again for display
func ShellifyPath(path string) string {
	home := os.Getenv("HOME") + "/"
	if strings.HasPrefix(path, home) {
		path = `~/` + path[len(home):]
	}
	return path
}

func getDefaultConfigPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", errors.New("no DP_CLI_CONFIG config file specified and failed to determine user's home directory")
	}
	return filepath.Join(usr.HomeDir, ".dp-cli-config.yml"), nil
}

// Dump returns the stringified config
func Dump() ([]byte, error) {
	c, err := Get()
	if err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetMyIP fetches your external IP address
func GetMyIP() (string, error) {
	if ip := os.Getenv("MY_IP"); len(ip) > 0 {
		isIP, err := regexp.Match(`^\d{1,3}(?:\.\d{1,3}){3}(?:/\d{1,2})?$`, []byte(ip))
		if err != nil {
			return "", err
		}
		if !isIP {
			return "", errors.New("unexpected format for var MY_IP")
		}
		return ip, nil
	}

	res, err := httpClient.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}

	defer func() {
		io.Copy(ioutil.Discard, res.Body)
	}()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code fetching IP: %d", res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// OverrideFrom merges prioritySvc into receiver, where non-nil
func (svc *Service) OverrideFrom(svcName string, prioritySvc Service) {
	if len(prioritySvc.Path) > 0 {
		svc.Path = prioritySvc.Path
	}
	if len(prioritySvc.InitCmds) > 0 {
		svc.InitCmds = append(svc.InitCmds, prioritySvc.InitCmds...)
	}
	if len(prioritySvc.LateInitCmds) > 0 {
		svc.LateInitCmds = append(svc.LateInitCmds, prioritySvc.LateInitCmds...)
	}
	if len(prioritySvc.StartCmd) > 0 {
		svc.StartCmd = prioritySvc.StartCmd
	}
	if len(prioritySvc.StopCmd) > 0 {
		svc.StopCmd = prioritySvc.StopCmd
	}
	if prioritySvc.Priority != nil {
		svc.Priority = prioritySvc.Priority
	}
	if len(prioritySvc.RepoURI) > 0 {
		svc.RepoURI = prioritySvc.RepoURI
	}
	if prioritySvc.Count != nil {
		svc.Count = prioritySvc.Count
	}
	if prioritySvc.Max != nil {
		svc.Max = prioritySvc.Max
	}
}

// WarnAt will warn if minLevel <= opts.Verbose
func (opts WithOpts) WarnAt(minLevel int, fmt string, fmtArgs ...interface{}) {
	if *opts.Verbose >= minLevel {
		out.WarnE(fmt, fmtArgs...)
	}
}

// WarnAtSvc will warn in particular format if minLevel <= opts.Verbose
func (opts WithOpts) WarnAtSvc(minLevel int, pre, svc interface{}, fmt string, fmtArgs ...interface{}) {
	if *opts.Verbose >= minLevel {
		fmtArgs = append([]interface{}{pre, svc}, fmtArgs...)
		out.WarnE("%-12s %-30q "+fmt, fmtArgs...)
	}
}
