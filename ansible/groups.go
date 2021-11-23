package ansible

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

var (
	groupPrefix = "["
	groupSuffix = "]"
)

// GetGroupsForEnvironment returns a list of ansible groups for the specified environment
func GetGroupsForEnvironment(dpSetUpPath, environment string) ([]string, error) {
	b, err := readFile(dpSetUpPath, environment)
	if err != nil {
		return nil, err
	}

	return parseFile(b)
}

func readFile(dpSetUpPath, environment string) ([]byte, error) {
	hostsPath := filepath.Join(dpSetUpPath, "ansible/inventories", environment, "hosts")
	return ioutil.ReadFile(hostsPath)
}

func parseFile(fileBytes []byte) ([]string, error) {
	var groups []string

	r := bufio.NewReader(bytes.NewReader(fileBytes))

	for {
		s, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		s = strings.TrimSpace(s)

		if isGroupLine(s) {
			name := s[1 : len(s)-1]
			groups = append(groups, name)
		}
	}

	return groups, nil
}

func isGroupLine(s string) bool {
	return strings.HasPrefix(s, groupPrefix) && strings.HasSuffix(s, groupSuffix)
}
