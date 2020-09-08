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

	"gopkg.in/yaml.v2"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Config struct {
	CMD          CMD                `yaml:"cmd"`
	Environments []Environment      `yaml:"environments"`
	SSHUser      string             `yaml:"ssh-user"`
	SourcePath   []string           `yaml:"dp-source-path"`
	Services     map[string]Service `yaml:"services"`
}

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

// Service allows individual configuration of a service
type Service struct {
	Path    string            `yaml:"path"`
	RepoURI string            `yaml:"repo_uri"`
	Once    bool              `yaml:"once"`
	Subnet  map[string]Subnet `yaml:"subnet"`
}

type Subnet struct {
	Ignore   bool   `yaml:"ignore"`
	Priority int    `yaml:"priority"`
	StartCmd string `yaml:"start-command"`
}

// WithOpts is used to pass cmdline opts to commands
type WithOpts struct {
	ForUser         *string
	HTTPOnly        *bool
	Interactive     *bool
	LimitWeb        *bool
	LimitPublishing *bool
	Verbose         *int
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

	// convert ~ to home dir in paths, or prepend SourceDir if relative
	for idx := range cfg.SourcePath {
		cfg.SourcePath[idx] = cfg.expandPath(cfg.SourcePath[idx])
	}
	for svcName, svc := range cfg.Services {
		svc.Path = cfg.FindOrFromURI(cfg.Services[svcName].Path, cfg.Services[svcName].RepoURI)
	}
	return &cfg, nil
}

func IsDir(dir string) (isDir bool, err error) {
	if s, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
	} else {
		isDir = s.IsDir()
	}
	return
}

func FindPath(dir string, path []string) string {
	for _, d := range path {
		fullPath := filepath.Join(d, dir)
		if isDir, err := IsDir(fullPath); err == nil && isDir {
			return fullPath
		}
	}
	return ""
}

func (cfg *Config) FindOrFromURI(path, uri string) string {
	if path == "" {
		if uri != "" {
			if lastSlashAt := strings.LastIndex(uri, "/"); lastSlashAt != -1 {
				path = uri[lastSlashAt:]
			}
		}
	}
	return cfg.expandPath(path)
}

func (cfg *Config) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		path = os.Getenv("HOME") + path[1:]
	}
	if !strings.HasPrefix(path, "/") {
		newPath := FindPath(path, cfg.SourcePath)
		if newPath != "" {
			path = newPath
		}

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
