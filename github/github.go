package github

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/out"
)

const (
	LATEST_RELEASE_URL         = "https://api.github.com/repos/ONSdigital/dp-cli/releases/latest"
	CURRENT_RELEASE_URL_PREFIX = "https://api.github.com/repos/ONSdigital/dp-cli/releases/tags"
	DATE_TIME_FORMAT           = "2006-01-02T15:04:05Z"
)

type release struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
}

type rateLimit struct {
	Message string `json:"message"`
}

func CheckApplicationVersion(version string) string {
	version = strings.TrimSuffix(version, "-dirty")

	latest := getRelease(LATEST_RELEASE_URL)
	latestBody := getReleaseBody(latest.Body)
	latestRelease := marshalBody(latestBody)

	checkForRateLimiting(latestRelease, latestBody)

	current := getRelease(fmt.Sprintf("%s/%s", CURRENT_RELEASE_URL_PREFIX, version))
	currentBody := getReleaseBody(current.Body)
	currentRelease := marshalBody(currentBody)

	currentVersion, _ := time.Parse(DATE_TIME_FORMAT, currentRelease.PublishedAt)
	latestVersion, _ := time.Parse(DATE_TIME_FORMAT, latestRelease.PublishedAt)

	if currentVersion != latestVersion {
		return fmt.Sprintf("Please update dp. Latest version: %s, your version: %s", latestRelease.TagName, currentRelease.TagName)
	}

	return ""
}

func getRelease(url string) *http.Response {
	release, err := http.Get(url)
	if err != nil {
		out.WarnFHighlight("github api call to get release failed: %s", err)
	}
	return release
}

func getReleaseBody(body io.Reader) []byte {
	returnBody, err := ioutil.ReadAll(body)
	if err != nil {
		out.WarnFHighlight("failed getting release body: %s", err)
	}
	return returnBody
}

func marshalBody(body []byte) release {
	var release release
	if err := json.Unmarshal(body, &release); err != nil {
		out.WarnFHighlight("failed unmarshaling release: %s", err)
	}
	return release
}

func checkForRateLimiting(appRelease release, body []byte) {
	emptyRelease := release{}
	if appRelease == emptyRelease {
		var rateLimited rateLimit
		if err := json.Unmarshal(body, &rateLimited); err != nil {
			out.WarnFHighlight("failed unmarshaling rate limit message: %s", err)
		}
		out.WarnFHighlight("rate limit exceeded: %s", rateLimited.Message)
	}
}
