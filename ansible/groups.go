package ansible

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// GetGroupsForEnvironment returns a list of ansible groups for the specified environment
func GetGroupsForEnvironment(dpSetUpPath, environment string) ([]string, error) {
	b, err := readFile(dpSetUpPath, environment)
	if err != nil {
		return nil, err
	}

	groups := parseFile(b)
	return groups, nil
}

func readFile(dpSetUpPath, environment string) ([]byte, error) {
	hostsPath := filepath.Join(dpSetUpPath, "ansible/inventories", environment, "hosts")
	return ioutil.ReadFile(hostsPath)
}

func parseFile(fileBytes []byte) []string {
	var groups []string

	r := bufio.NewReader(bytes.NewReader(fileBytes))

	for {
		s, err := r.ReadString('\n')
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "[") && strings.Contains(s, ":") && strings.HasSuffix(s, "]") {
			name := s[1:strings.Index(s, ":")]
			groups = append(groups, name)
		}
		if err == io.EOF {
			break
		}
	}
	return groups
}
