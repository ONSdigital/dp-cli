package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/project_generation"
	"gopkg.in/yaml.v2"
)

// tags refer to dp-cli-config.yml environment tags which put that environment into group types
const (
	TAG_AWSA   = "awsa"   // legacy/deprecated
	TAG_CI     = "ci"     // concourse
	TAG_LIVE   = "live"   // production
	TAG_SECURE = "secure" // has secure data (e.g. prod, staging)
	TAG_NISRA  = "nisra"  // NISRA
)

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

type Config struct {
	CMD                    CMD           `yaml:"cmd"`
	EKS                    EKS           `yaml:"eks"`
	Environments           []Environment `yaml:"environments"`
	SSHUser                *string       `yaml:"ssh-user"`
	UserName               *string       `yaml:"user-name"`
	IPAddress              *string       `yaml:"ip-address"`
	IPv4Address            *string       `yaml:"ipv4-address"`
	IPv6Address            *string       `yaml:"ipv6-address"`
	HttpOnly               *bool         `yaml:"http-only"`
	DPSetupPath            string        `yaml:"dp-setup-path"`
	NisraPath              string        `yaml:"dp-nisra-path"`
	DPCIPath               string        `yaml:"dp-ci-path"`
	DPHierarchyBuilderPath string        `yaml:"dp-hierarchy-builder-path"`
	DPCodeListScriptsPath  string        `yaml:"dp-code-list-scripts-path"`
	DPCLIPath              string        `yaml:"dp-cli-path"`
}

// EKS holds optional overrides for EKS tunnel discovery tags
type EKS struct {
	TunnelBoxRoleTag string `yaml:"tunnel-box-role-tag"`
	ClusterAccessTag string `yaml:"cluster-access-tag"`
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
	SSHUser    string     `yaml:"ssh-user"`
	Tags       []string   `yaml:"tags"`
	ExtraPorts ExtraPorts `yaml:"extra-ports"`
}

// ExtraPorts is a list of ports for the given Security Group
type ExtraPorts struct {
	Bastion    []int32 `yaml:"bastion"`
	Publishing []int32 `yaml:"publishing"`
	Web        []int32 `yaml:"web"`
}

// Get returns the config struct by parsing the YML file
func Get() (*Config, error) {
	path := getConfigPath()

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse %q: %w", path, err)
	}

	cfg.expandPaths()

	// if compile-time templatePath does not exist, or dp-cli-path set in config
	if _, err = os.Stat(project_generation.GetTemplatePath()); os.IsNotExist(err) || cfg.DPCLIPath != "" {
		if cfg.DPCLIPath != "" {
			project_generation.SetTemplatePath(cfg.DPCLIPath + "/project_generation/content/templates")
		}
	}

	return &cfg, nil
}

func (cfg *Config) expandPaths() {
	cfg.DPCIPath = expandPath(cfg.DPCIPath)
	cfg.DPHierarchyBuilderPath = expandPath(cfg.DPHierarchyBuilderPath)
	cfg.DPSetupPath = expandPath(cfg.DPSetupPath)
	cfg.NisraPath = expandPath(cfg.NisraPath)
	cfg.DPCodeListScriptsPath = expandPath(cfg.DPCodeListScriptsPath)
	cfg.DPCLIPath = expandPath(cfg.DPCLIPath)
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

func (cfg Config) checkGotIP(ip string) (bool, error) {
	return regexp.MatchString(`^\d{1,3}(?:\.\d{1,3}){3}(?:/\d{1,2})?$`, ip)
}

// GetMyIP returns first IP in: `--ip` flag, `MY_IP` env var, config file, external service
func (cfg Config) GetMyIP() (string, error) {
	// flag or config-file used?
	if cfg.IPAddress != nil && len(*cfg.IPAddress) > 0 {
		if isValidIP, err := cfg.checkGotIP(*cfg.IPAddress); err != nil || !isValidIP {
			return "", fmt.Errorf("unexpected format for IP (from --ip flag or config-file): %w", err)
		}
		return *cfg.IPAddress, nil
	}

	// env var used?
	if ip := os.Getenv("MY_IP"); len(ip) > 0 {
		if isValidIP, err := cfg.checkGotIP(ip); err != nil || !isValidIP {
			return "", fmt.Errorf("unexpected format for env var MY_IP: %w", err)
		}
		return ip, nil
	}

	// use remote service to obtain IP
	res, err := httpClient.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("cannot get IP from service (consider using `--ip` flag instead): %w", err)
	}

	defer func() {
		res.Body.Close()
	}()

	if res.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code fetching IP (consider using `--ip` flag instead): %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if isValidIP, err := cfg.checkGotIP(string(b)); err != nil || !isValidIP {
		return "", fmt.Errorf("unexpected format for IP result from IP service: %w", err)
	}

	return string(b), nil
}

// GetMyIPs2 returns both IPv4 and IPv6 addresses, checking config > cli > env, then external service if needed.
func (cfg Config) GetMyIPs2() (ipv4 string, ipv6 string, err error) {
	// 1. Check config/env for explicit values
	if cfg.IPv4Address != nil && len(*cfg.IPv4Address) > 0 {
		ipv4 = *cfg.IPv4Address
	} else if cfg.IPAddress != nil && len(*cfg.IPAddress) > 0 { // legacy fallback
		ipv4 = *cfg.IPAddress
	} else if ip := os.Getenv("MY_IPV4"); len(ip) > 0 {
		ipv4 = ip
	}

	if cfg.IPv6Address != nil && len(*cfg.IPv6Address) > 0 {
		ipv6 = *cfg.IPv6Address
	} else if ip := os.Getenv("MY_IPV6"); len(ip) > 0 {
		ipv6 = ip
	}

	// 2. If not set, fetch from external service
	if ipv4 == "" {
		res, err4 := httpClient.Get("https://api.ipify.org")
		if err4 == nil && res.StatusCode == 200 {
			b, errRead := io.ReadAll(res.Body)
			res.Body.Close()
			if errRead == nil {
				s := string(b)
				if isValidIP, _ := cfg.checkGotIP(s); isValidIP {
					ipv4 = s
				}
			}
		}
	}
	if ipv6 == "" {
		res, err6 := httpClient.Get("https://api64.ipify.org")
		if err6 == nil && res.StatusCode == 200 {
			b, errRead := io.ReadAll(res.Body)
			res.Body.Close()
			if errRead == nil {
				s := string(b)
				// TODO: improve IPv6 validation
				if len(s) > 0 {
					ipv6 = s
				}
			}
		}
	}

	// 3. Validate at least one IP found
	if ipv4 == "" && ipv6 == "" {
		err = fmt.Errorf("could not determine IPv4 or IPv6 address from config, env, or external service")
	}
	return
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

func (cfg Config) IsAWSA(env string) bool {
	return cfg.hasTag(env, TAG_AWSA)
}
func (env Environment) IsAWSA() bool {
	return env.hasTag(TAG_AWSA)
}
func (cfg Config) IsCI(env string) bool {
	return cfg.hasTag(env, TAG_CI)
}
func (env Environment) IsCI() bool {
	return env.hasTag(TAG_CI)
}
func (cfg Config) IsLive(env string) bool {
	return cfg.hasTag(env, TAG_LIVE)
}
func (env Environment) IsLive() bool {
	return env.hasTag(TAG_LIVE)
}
func (cfg Config) IsNisra(env string) bool {
	return cfg.hasTag(env, TAG_NISRA)
}
func (env Environment) IsNisra() bool {
	return env.hasTag(TAG_NISRA)
}
func (cfg Config) IsSecure(env string) bool {
	return cfg.hasTag(env, TAG_SECURE)
}
func (env Environment) IsSecure() bool {
	return env.hasTag(TAG_SECURE)
}

func (cfg Config) GetProfile(env string) string {
	for _, e := range cfg.Environments {
		if e.Name == env {
			if e.Profile != "" {
				return e.Profile
			}
			return env
		}
	}
	return "noEnv"
}

func (cfg Config) GetPath(env Environment) string {
	if env.IsCI() {
		return cfg.DPCIPath
	}
	if env.IsNisra() {
		return cfg.NisraPath
	}
	return cfg.DPSetupPath
}

func (cfg Config) GetAnsibleDirectory(env Environment) string {
	if env.IsCI() {
		return filepath.Join(cfg.DPCIPath, "ansible")
	}
	if env.IsNisra() {
		return filepath.Join(cfg.NisraPath, "ansible")
	}
	return filepath.Join(cfg.DPSetupPath, "ansible")
}
