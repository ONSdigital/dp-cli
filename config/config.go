package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// tags refer to dp-cli-config.yml environment tags which put that environment into group types
const (
	TAG_AWSB = "awsb"
	TAG_CI   = "ci"
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Config struct {
	CMD                    CMD           `yaml:"cmd"`
	Environments           []Environment `yaml:"environments"`
	User                   *string       `yaml:"ssh-user"`
	IPAddress              *string       `yaml:"ip-address"`
	HttpOnly               *bool         `yaml:"http-only"`
	DPSetupPath            string        `yaml:"dp-setup-path"`
	DPCIPath               string        `yaml:"dp-ci-path"`
	DPHierarchyBuilderPath string        `yaml:"dp-hierarchy-builder-path"`
	DPCodeListScriptsPath  string        `yaml:"dp-code-list-scripts-path"`
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
	User       string     `yaml:"user"`
	Tags       []string   `yaml:"tags"`
	ExtraPorts ExtraPorts `yaml:"extra-ports"`
}

// ExtraPorts is a list of ports for the given Security Group
type ExtraPorts struct {
	Bastion    []int64 `yaml:"bastion"`
	Publishing []int64 `yaml:"publishing"`
	Web        []int64 `yaml:"web"`
}

// Get returns the config struct by parsing the YML file
func Get() (*Config, error) {
	path := getConfigPath()

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse %q: %w", path, err)
	}

	cfg.expandPaths()

	return &cfg, nil
}

func (cfg *Config) expandPaths() {
	cfg.DPCIPath = expandPath(cfg.DPCIPath)
	cfg.DPHierarchyBuilderPath = expandPath(cfg.DPHierarchyBuilderPath)
	cfg.DPSetupPath = expandPath(cfg.DPSetupPath)
	cfg.DPCodeListScriptsPath = expandPath(cfg.DPCodeListScriptsPath)
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		path = strings.Replace(path, "~", "${HOME}", 1)
	}
	path = os.ExpandEnv(path)
	return path
}

func getConfigPath() (path string) {
	path = os.Getenv("DP_CLI_CONFIG")
	if len(path) == 0 {
		path = expandPath("~/.dp-cli-config.yml")
	}
	return
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

func (cfg Config) checkGotIP() (bool, error) {
	return regexp.MatchString(`^\d{1,3}(?:\.\d{1,3}){3}(?:/\d{1,2})?$`, *cfg.IPAddress)
}

// GetMyIP fetches your external IP address
func (cfg Config) GetMyIP() (string, error) {
	if cfg.IPAddress == nil {
		s := ""
		cfg.IPAddress = &s
	}

	// flag used?
	if len(*cfg.IPAddress) > 0 {
		if isIP, err := cfg.checkGotIP(); err != nil || !isIP {
			return "", fmt.Errorf("unexpected IP format for flag: %w", err)
		}
		return *cfg.IPAddress, nil
	}

	// env var used?
	if *cfg.IPAddress = os.Getenv("MY_IP"); len(*cfg.IPAddress) > 0 {
		if isIP, err := cfg.checkGotIP(); err != nil || !isIP {
			return "", fmt.Errorf("unexpected format for var MY_IP: %w", err)
		}
		return *cfg.IPAddress, nil
	}

	// use remote service to obtain IP
	res, err := httpClient.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("cannot get IP from service: %w", err)
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

func (env Environment) hasTag(tag string) bool {
	for _, eachTag := range env.Tags {
		if eachTag == tag {
			return true
		}
	}
	return false
}

func (cfg Config) hasTag(env, tag string) bool {
	for _, e := range cfg.Environments {
		if e.Name == env {
			return e.hasTag(tag)
		}
	}
	return false
}

func (cfg Config) IsAWSB(env string) bool {
	return cfg.hasTag(env, TAG_AWSB)
}
func (env Environment) IsAWSB() bool {
	return env.hasTag(TAG_AWSB)
}
func (cfg Config) IsCI(env string) bool {
	return cfg.hasTag(env, TAG_CI)
}
func (env Environment) IsCI() bool {
	return env.hasTag(TAG_CI)
}

func (cfg Config) GetPath(env Environment) string {
	if env.IsCI() {
		return cfg.DPCIPath
	}
	return cfg.DPSetupPath
}

func (cfg Config) GetAnsibleDirectory(env Environment) string {
	if env.IsCI() {
		return filepath.Join(cfg.DPCIPath, "ansible")
	}
	return filepath.Join(cfg.DPSetupPath, "ansible")
}
