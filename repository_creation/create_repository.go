package repository_creation

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ONSdigital/dp-cli/project_generation"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/google/go-github/v28/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	org = "ONSdigital"
	// Team: "DigitalPublishing", slug: "digitalpublishing", id: 779417
	dpTeamSlug = "DigitalPublishing"
	teamID     = int64(779417)
	mainBranch = "main"
)

func RunGenerateRepo(cmd *cobra.Command, args []string) error {
	var err error
	nameOfApp, _ := cmd.Flags().GetString("name")
	token, _ := cmd.Flags().GetString("token")
	branchStrategyInput, _ := cmd.Flags().GetString("strategy")
	branchStrategy := strings.ToLower(strings.TrimSpace(branchStrategyInput))
	_, err = GenerateGithub(nameOfApp, "", "", token, branchStrategy)
	if err != nil {
		return err
	}

	return nil
}

// GenerateGithub is the entry point to generating the repository
func GenerateGithub(name, description string, ProjectType project_generation.ProjectType, personalAccessToken string, branchStrategy string) (cloneUrl string, err error) {
	accessToken, repoName, repoDescription, defaultBranch := getConfigurationsForNewRepo(name, description, ProjectType, personalAccessToken, branchStrategy)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	hasAccess, err := checkAccess(ctx, client)
	if err != nil {
		log.Error(ctx, "failed to check if had access", err)
		return "", err
	}
	if !hasAccess {
		log.Error(ctx, "user does not have access", err)
		return cloneUrl, err
	}

	repo := &github.Repository{
		Name:          github.String(repoName),
		Description:   github.String(repoDescription),
		DefaultBranch: github.String(defaultBranch),
		MasterBranch:  github.String(mainBranch),
		Private:       github.Bool(false),
		HasWiki:       github.Bool(false),
		HasProjects:   github.Bool(false),
		AutoInit:      github.Bool(true),
	}

	err = createRepo(ctx, client, repo)
	if err != nil {
		log.Error(ctx, "unable to create repository", err)
		return cloneUrl, err
	}

	if ProjectType != "generic-project" || branchStrategy == "git" {
		err = createDevelopBranch(ctx, client, repoName)
		if err != nil {
			log.Error(ctx, "unable to create develop branch", err)
			return cloneUrl, err
		}
	}

	err = setDevelopAsDefaultBranch(ctx, client, repoName, repo)
	if err != nil {
		log.Error(ctx, "failed to set default branch to develop", err)
		return cloneUrl, err
	}

	err = setBranchProtections(ctx, client, repoName, branchStrategy)
	if err != nil {
		log.Error(ctx, "unable to set all branch protections", err)
		return cloneUrl, err
	}

	err = setTeamsAndCollaborators(ctx, client, repoName)
	if err != nil {
		log.Error(ctx, "unable to set team and collaborators", err)
		return cloneUrl, err
	}
	repositoryObj, _, err := client.Repositories.Get(ctx, org, repoName)
	if err != nil {
		log.Error(ctx, "unable to locate the the attempted newly created repository", err)
		return cloneUrl, err
	}
	cloneUrl = repositoryObj.GetCloneURL()
	// Notify user of completion and get them to turn off actions
	log.Info(ctx, "repository has successfully been create please Disable Actions for this repository")
	return cloneUrl, nil
}

// setTeamsAndCollaborators will set the DigitalPublishing team as a team working on the repo and removes the creator from being a collaborator
func setTeamsAndCollaborators(ctx context.Context, client *github.Client, repoName string) error {
	addTeamRepoOptions := github.TeamAddTeamRepoOptions{Permission: "admin"}
	_, err := client.Teams.AddTeamRepo(ctx, teamID, org, repoName, &addTeamRepoOptions)
	if err != nil {
		log.Error(ctx, "unable to add collaborators", err)
		return err
	}

	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Error(ctx, "unable to get current github user", err, log.Data{"response": resp})
	}
	userHandle := *user.Login

	resp, err = client.Repositories.RemoveCollaborator(ctx, org, repoName, userHandle)
	if err != nil {
		log.Error(ctx, "unable to remove self as a collaborator", err, log.Data{"response": resp})
	}
	return err
}

// setBranchProtections sets the protections for both main and develop branches
func setBranchProtections(ctx context.Context, client *github.Client, repoName, branchStrategy string) error {
	requiredStatusChecks := github.RequiredStatusChecks{
		Strict:   true,
		Contexts: []string{},
	}

	dismissalRestrictionsRequest := github.DismissalRestrictionsRequest{
		Users: &[]string{},
		Teams: &[]string{dpTeamSlug},
	}
	requiredPullRequestReviewsEnforcementRequest := github.PullRequestReviewsEnforcementRequest{
		DismissalRestrictionsRequest: &dismissalRestrictionsRequest,
		DismissStaleReviews:          true,
		RequireCodeOwnerReviews:      true,
		RequiredApprovingReviewCount: 1,
	}

	branchRestrictions := github.BranchRestrictionsRequest{
		Users: []string{},
		Teams: []string{dpTeamSlug},
	}

	protectionRequest := github.ProtectionRequest{
		RequiredStatusChecks:       &requiredStatusChecks,
		RequiredPullRequestReviews: &requiredPullRequestReviewsEnforcementRequest,
		EnforceAdmins:              true,
		Restrictions:               &branchRestrictions,
	}
	_, _, err := client.Repositories.UpdateBranchProtection(ctx, org, repoName, "main", &protectionRequest)
	if err != nil {
		log.Error(ctx, "update branch protection failed for main", err)
		return err
	}

	if branchStrategy == "git" {
		_, _, err = client.Repositories.UpdateBranchProtection(ctx, org, repoName, "develop", &protectionRequest)
		if err != nil {
			log.Error(ctx, "update branch protection failed for develop", err)
			return err
		}
	}
	var resp *github.Response
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "main")
	if err != nil {
		log.Error(ctx, "adding protection, require signatures failed on branch main", err, log.Data{"response": resp})
		return err
	}
	if branchStrategy == "git" {
		_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "develop")
		if err != nil {
			log.Error(ctx, "adding protection, require signatures failed on branch develop", err, log.Data{"response": resp})
			return err
		}
	}
	return err
}

// setDevelopAsDefaultBranch will set the develop branch as the repositories default branch
func setDevelopAsDefaultBranch(ctx context.Context, client *github.Client, repoName string, repo *github.Repository) error {
	repo.AutoInit = nil
	_, resp, err := client.Repositories.Edit(ctx, org, repoName, repo)
	if err != nil {
		log.Error(ctx, "failed to set develop as the default branch", err, log.Data{"response": resp})
	}
	return err
}

// createDevelopBranch will create a new branch named /develop, based on the last ref on main branch
func createDevelopBranch(ctx context.Context, client *github.Client, repoName string) error {
	ref, resp, err := client.Git.GetRef(ctx, org, repoName, "heads/main")
	if err != nil {
		log.Error(ctx, "get reference to main commit failed", err, log.Data{"response": resp})
		return err
	}
	developBranch := "heads/develop"
	ref.Ref = &developBranch

	_, resp, err = client.Git.CreateRef(ctx, org, repoName, ref)
	if err != nil {
		log.Error(ctx, "create Reference to new develop branch failed", err, log.Data{"response": resp})
		return err
	}
	return nil
}

// createRepo makes a call via the gitHubAPI to generate a new repository
func createRepo(ctx context.Context, client *github.Client, repo *github.Repository) error {
	newRepo, _, err := client.Repositories.Create(ctx, org, repo)
	fmt.Println(newRepo)
	if err != nil {
		log.Error(ctx, "repo creation failed", err)
		return err
	}
	return nil
}

// getConfigurationsForNewRepo gets required configuration information from the end user
func getConfigurationsForNewRepo(name, description string, projType project_generation.ProjectType, personalAccessToken string, branchStrategy string) (accessToken, repoName, repoDescription, defaultBranch string) {
	defaultBranch = "develop"
	if personalAccessToken == "" {
		token, exists := os.LookupEnv("GITHUB_PERSONAL_ACCESS_TOKEN")
		if exists {
			accessToken = token
		} else {
			accessToken = PromptForInput("Please provide your personal access token (to create one follow this guide https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token):")
		}
	} else {
		accessToken = personalAccessToken
	}
	if name == "" || name == "unset" {
		repoName = PromptForInput("Please provide the full name for the new repository (note 'unset' is not an applicable name'):")
	} else {
		repoName = name
	}
	if description == "" {
		repoDescription = PromptForInput("Please provide a description for the repository:")
	} else {
		repoDescription = description
	}
	if branchStrategy == "" {
		prompt := "Please pick the branching strategy you wish this repo to use:"
		options := []string{"github flow", "git flow"}
		ctx := context.Background()
		branchStrategy, err := project_generation.OptionPromptInput(ctx, prompt, options...)
		if err != nil {
			log.Error(ctx, "error getting branch strategy", err)
		}
		branchStrategy = strings.Replace(branchStrategy, " flow", "", -1)
	}
	if projType == "generic-project" || branchStrategy == "github" {
		defaultBranch = "main"
	}
	return accessToken, repoName, repoDescription, defaultBranch
}

// PromptForInput gives a user a message and expect input to be provided
func PromptForInput(prompt string) string {
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(prompt)
	scanner.Scan()
	input = scanner.Text()
	if scanner.Err() != nil {
		ctx := context.Background()
		log.Error(ctx, "repo creation failed", scanner.Err())
	}
	return input
}

// checkAccess ensures the user has provided a a valid access token
func checkAccess(ctx context.Context, client *github.Client) (bool, error) {
	_, _, err := client.Repositories.List(ctx, "", nil)
	if err != nil {
		log.Error(ctx, "failed to get list of repos", err)
		return false, err
	}
	return true, nil
}
