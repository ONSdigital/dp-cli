package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ONSdigital/dp-cli/out"

	"github.com/spf13/cobra"
)

type Release struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
}

type Tag struct {
	Ref    string    `json:"ref"`
	Object TagObject `json:"object"`
}

type TagObject struct {
	Url string `json:"url"`
}

type Tags struct {
	Object TagObject `json:"object"`
}

type RateLimit struct {
	Message string `json:"message"`
}

func GithubCall(version string) string {

	version = strings.TrimSuffix(version, "-dirty")

	latest, err := http.Get("https://api.github.com/repos/ONSdigital/dp-cli/releases/latest")
	if err != nil {
		out.WarnFHighlight("github api call to get latest release failed: %s", err)
	}

	latestBody, err := ioutil.ReadAll(latest.Body)
	if err != nil {
		out.WarnFHighlight("failed getting latest release body: %s", err)
	}

	var latestRelease Release
	if err := json.Unmarshal(latestBody, &latestRelease); err != nil {
		out.WarnFHighlight("failed unmarshaling latest release: %s", err)
	}

	emptyRelease := Release{}
	if latestRelease == emptyRelease {
		var rateLimited RateLimit
		if err := json.Unmarshal(latestBody, &rateLimited); err != nil {
			out.WarnFHighlight("failed unmarshaling rate limit message: %s", err)
		}
		out.WarnFHighlight("rate limit exceeded: %s", rateLimited.Message)
	}

	current, err := http.Get(fmt.Sprintf("https://api.github.com/repos/ONSdigital/dp-cli/releases/tags/%s", "v0.38.0"))
	if err != nil {
		out.WarnFHighlight("github api call to get current release failed: %s", err)
	}

	currentBody, err := ioutil.ReadAll(current.Body)
	if err != nil {
		out.WarnFHighlight("failed getting current release body: %s", err)
	}

	var currentRelease Release
	if err := json.Unmarshal(currentBody, &currentRelease); err != nil {
		out.WarnFHighlight("failed unmarshaling current release: %s", err)
	}

	layout := "2006-01-02T15:04:05Z"
	currentVersion, _ := time.Parse(layout, currentRelease.PublishedAt)
	latestVersion, _ := time.Parse(layout, latestRelease.PublishedAt)

	if currentVersion != latestVersion {
		return fmt.Sprintf("Please update dp. Latest version: %s, your version: %s", latestRelease.TagName, currentRelease.TagName)
	}

	return ""
}

func versionSubCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the app version",
		Run: func(cmd *cobra.Command, args []string) {
			out.Info(appVersion)
		},
	}
}
