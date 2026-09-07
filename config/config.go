package config

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
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
	CMD                    CMD               `yaml:"cmd"`
	EKS                    EKS               `yaml:"eks"`
	Environments           []Environment     `yaml:"environments"`
	ProfileSuffixes        map[string]string `yaml:"profile-suffixes"`
	CommandPrivileges      map[string]string `yaml:"command-privileges"`
	SSHUser                *string           `yaml:"ssh-user"`
	UserName               *string           `yaml:"user-name"`
	IPAddress              *string           `yaml:"ip-address"`
	IPv4Address            *string           `yaml:"ipv4-address"`
	IPv6Address            *string           `yaml:"ipv6-address"`
	HttpOnly               *bool             `yaml:"http-only"`
	DPSetupPath            string            `yaml:"dp-setup-path"`
	NisraPath              string            `yaml:"dp-nisra-path"`
	DPCIPath               string            `yaml:"dp-ci-path"`
	DPHierarchyBuilderPath string            `yaml:"dp-hierarchy-builder-path"`
	DPCodeListScriptsPath  string            `yaml:"dp-code-list-scripts-path"`
	DPCLIPath              string            `yaml:"dp-cli-path"`
}

// EKS holds optional overrides for EKS tunnel discovery and local tunnel state.
type EKS struct {
	TunnelBoxRoleTag string `yaml:"tunnel-box-role-tag"`
	ClusterAccessTag string `yaml:"cluster-access-tag"`
	// StateDir is where per-cluster tunnel state files are stored. Defaults to
	// /tmp/eks-tunnels-ssm when empty (ephemeral, cleared on reboot).
	StateDir string `yaml:"state-dir"`
	// BasePort and MaxPort bound the local port range used for SSM
	// port-forwards. Defaults to 9443 and 9500 (inclusive) when unset.
	BasePort int `yaml:"base-port"`
	MaxPort  int `yaml:"max-port"`
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
	cfg.EKS.StateDir = expandPath(cfg.EKS.StateDir)
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
	if path == "" {
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

// IsValidIPv4 validates an IPv4 address (with optional CIDR suffix) using the standard library.
func IsValidIPv4(ip string) bool {
	// Try as a CIDR prefix first (e.g. "1.2.3.4/32")
	if prefix, err := netip.ParsePrefix(ip); err == nil {
		return prefix.Addr().Is4()
	}
	// Try as a plain address
	if addr, err := netip.ParseAddr(ip); err == nil {
		return addr.Is4()
	}
	return false
}

// IsValidIPv6 validates an IPv6 address (with optional CIDR suffix) using the standard library.
func IsValidIPv6(ip string) bool {
	// Try as a CIDR prefix first (e.g. "2001:db8::/64")
	if prefix, err := netip.ParsePrefix(ip); err == nil {
		return prefix.Addr().Is6()
	}
	// Try as a plain address
	if addr, err := netip.ParseAddr(ip); err == nil {
		return addr.Is6()
	}
	return false
}

// GetMyIP returns first IP in: `--ip` flag, `MY_IP` env var, config file, external service
func (cfg Config) GetMyIP() (string, error) {
	// flag or config-file used?
	if cfg.IPAddress != nil && *cfg.IPAddress != "" {
		if !IsValidIPv4(*cfg.IPAddress) {
			return "", fmt.Errorf("unexpected format for IP (from --ip flag or config-file): %q", *cfg.IPAddress)
		}
		return *cfg.IPAddress, nil
	}

	// env var used?
	if ip := os.Getenv("MY_IP"); ip != "" {
		if !IsValidIPv4(ip) {
			return "", fmt.Errorf("unexpected format for env var MY_IP: %q", ip)
		}
		return ip, nil
	}

	// use remote service to obtain IP
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.ipify.org", http.NoBody)
	if err != nil {
		return "", fmt.Errorf("cannot create request for IP service: %w", err)
	}
	res, err := httpClient.Do(req)
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

	ip := string(b)
	if !IsValidIPv4(ip) {
		return "", fmt.Errorf("unexpected format for IP result from IP service: %q", ip)
	}

	return ip, nil
}

// GetMyIPs returns both IPv4 and IPv6 addresses, checking config > env, then external service if needed.
func (cfg Config) GetMyIPs() (ipv4, ipv6 string, err error) {
	// 1. Check config/env for explicit values
	if cfg.IPv4Address != nil && *cfg.IPv4Address != "" {
		ipv4 = *cfg.IPv4Address
	} else if cfg.IPAddress != nil && *cfg.IPAddress != "" { // legacy fallback
		ipv4 = *cfg.IPAddress
	} else if ip := os.Getenv("MY_IPV4"); ip != "" {
		ipv4 = ip
	}

	if cfg.IPv6Address != nil && *cfg.IPv6Address != "" {
		ipv6 = *cfg.IPv6Address
	} else if ip := os.Getenv("MY_IPV6"); ip != "" {
		ipv6 = ip
	}

	// 2. Validate any explicitly-provided values
	if ipv4 != "" && !IsValidIPv4(ipv4) {
		return "", "", fmt.Errorf("invalid IPv4 address from config/env: %q", ipv4)
	}
	if ipv6 != "" && !IsValidIPv6(ipv6) {
		return "", "", fmt.Errorf("invalid IPv6 address from config/env: %q", ipv6)
	}

	// 3. If not set, fetch from external service
	if ipv4 == "" {
		req4, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.ipify.org", http.NoBody)
		res, err4 := httpClient.Do(req4)
		if err4 == nil && res.StatusCode == 200 {
			b, errRead := io.ReadAll(res.Body)
			res.Body.Close()
			if errRead == nil {
				s := strings.TrimSpace(string(b))
				if IsValidIPv4(s) {
					ipv4 = s
				}
			}
		}
	}
	if ipv6 == "" {
		// api6.ipify.org is an IPv6-only endpoint (AAAA records only).
		// If the user has no IPv6 connectivity, the request will fail
		// (DNS resolution or connection error), which cleanly indicates
		// "no IPv6 available" rather than silently returning an IPv4.
		req6, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api6.ipify.org", http.NoBody)
		res, err6 := httpClient.Do(req6)
		if err6 == nil && res.StatusCode == 200 {
			b, errRead := io.ReadAll(res.Body)
			res.Body.Close()
			if errRead == nil {
				s := strings.TrimSpace(string(b))
				if IsValidIPv6(s) {
					ipv6 = s
				}
			}
		}
	}

	// 4. Validate at least one IP found
	if ipv4 == "" && ipv6 == "" {
		err = fmt.Errorf("could not determine IPv4 or IPv6 address from config, env, or external service")
	}
	return ipv4, ipv6, err
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
	//nolint:gocritic // rangeValCopy acceptable for environment config iteration
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

//nolint:gocritic // rangeValCopy acceptable for environment config iteration
func (cfg Config) GetProfile(env string) string {
	for _, e := range cfg.Environments { //nolint:gocritic // rangeValCopy acceptable for config iteration
		if e.Name == env {
			if e.Profile != "" {
				return e.Profile
			}
			return env
		}
	}
	return "noEnv"
}

// GetProfileForCommand resolves the profile for a given environment and command path.
// It walks from most specific to least specific command path (e.g. "eks.session.start" → "eks.session" → "eks").
// If no match is found or profile-suffixes is not configured, returns the base profile (backwards compatible).
func (cfg Config) GetProfileForCommand(env, commandPath string) string {
	base := cfg.GetProfile(env)

	if len(cfg.ProfileSuffixes) == 0 || len(cfg.CommandPrivileges) == 0 {
		return base
	}

	// Walk from most specific to least specific
	path := commandPath
	for path != "" {
		if priv, ok := cfg.CommandPrivileges[path]; ok {
			if suffix, ok := cfg.ProfileSuffixes[priv]; ok {
				return base + suffix
			}
			return base
		}
		// Remove last segment
		lastDot := strings.LastIndex(path, ".")
		if lastDot < 0 {
			break
		}
		path = path[:lastDot]
	}

	return base
}

// GetProfileWithRole resolves the profile for a given environment and explicit role override.
// Used when a user passes -view, -engineer, or -admin flags.
func (cfg Config) GetProfileWithRole(env, role string) string {
	base := cfg.GetProfile(env)
	if role == "" || len(cfg.ProfileSuffixes) == 0 {
		return base
	}
	if suffix, ok := cfg.ProfileSuffixes[role]; ok {
		return base + suffix
	}
	return base
}

// GetRoleSuffix returns the suffix for a given role name, or empty string if not configured.
func (cfg Config) GetRoleSuffix(role string) string {
	if suffix, ok := cfg.ProfileSuffixes[role]; ok {
		return suffix
	}
	return ""
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
