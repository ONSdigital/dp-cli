package repository

import (
	"bufio"
	"context"
	projectgeneration "dp-utils/app-generation"
	"fmt"
	"github.com/ONSdigital/log.go/log"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
	"os"
)

const (
	org = "ONSdigital"
	// Team: "DigitalPublishing", slug: "digitalpublishing", id: 779417
	dpTeamSlug   = "DigitalPublishing"
	teamID       = int64(779417)
	masterBranch = "master"
)

// GenerateGithub is the entry point to generating the repository
func GenerateGithub(name string, ProjectType projectgeneration.ProjectType, personalAccessToken string, isLibrary bool) (cloneUrl string, err error) {
	fmt.Println("This script will create a new ONS Digital Publishing repository." +
		"In order to create and configure a new repository please answer the prompts.")

	accessToken, userHandle, repoName, repoDescription, defaultBranch := getConfigurationsForNewRepo(name, ProjectType, personalAccessToken, isLibrary)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	hasAccess, err := checkAccess(ctx, client)
	if !hasAccess {
		log.Event(ctx, "user does not have access", log.Error(err))
		return cloneUrl, err
	}

	repo := &github.Repository{
		Name:          github.String(repoName),
		Description:   github.String(repoDescription),
		DefaultBranch: github.String(defaultBranch),
		MasterBranch:  github.String(masterBranch),
		Private:       github.Bool(false),
		HasWiki:       github.Bool(false),
		HasProjects:   github.Bool(false),
		AutoInit:      github.Bool(true),
	}

	err = createRepo(ctx, client, repo)
	if err != nil {
		log.Event(ctx, "unable to create repository", log.Error(err))
		return cloneUrl, err
	}

	err = createDevelopBranch(ctx, client, repoName)
	if err != nil {
		log.Event(ctx, "unable to create develop branch", log.Error(err))
		return cloneUrl, err
	}

	err = setDevelopAsDefaultBranch(ctx, client, repoName, repo)
	if err != nil {
		log.Event(ctx, "failed to set default branch to develop", log.Error(err))
		return cloneUrl, err
	}

	err = setBranchProtections(ctx, client, repoName)
	if err != nil {
		log.Event(ctx, "unable to set all branch protections, please review and correct these manually", log.Error(err))
		return cloneUrl, err
	}

	err = setTeamsAndCollaborators(ctx, client, repoName, userHandle)
	if err != nil {
		log.Event(ctx, "unable to set team and collaborators", log.Error(err))
		return cloneUrl, err
	}
	repositoryObj, _, err := client.Repositories.Get(ctx, org, repoName)
	if err != nil {
		log.Event(ctx, "unable to locate the the attempted newly created repository", log.Error(err))
		return cloneUrl, err
	}
	cloneUrl = repositoryObj.GetCloneURL()
	fmt.Println("cloneUrl is: " + cloneUrl)
	// Notify user of completion and get them to turn off actions
	log.Event(ctx, "repository has successfully been create please Disable Actions for this repository")
	return cloneUrl, nil
}

// setTeamsAndCollaborators will set the DigitalPublishing team as a team working on the repo and removes the creator from being a collaborator
func setTeamsAndCollaborators(ctx context.Context, client *github.Client, repoName string, userHandle string) error {
	addTeamRepoOptions := github.TeamAddTeamRepoOptions{Permission: "admin"}
	resp, err := client.Teams.AddTeamRepo(ctx, teamID, org, repoName, &addTeamRepoOptions)
	if err != nil {
		log.Event(ctx, "unable to add collaborators", log.Error(err))
	}

	resp, err = client.Repositories.RemoveCollaborator(ctx, org, repoName, userHandle)
	if err != nil {
		log.Event(ctx, "unable to remove self as a collaborator", log.Error(err), log.Data{"response": resp})
	}
	return err
}

// setBranchProtections sets the protections for both master and develop branches
func setBranchProtections(ctx context.Context, client *github.Client, repoName string) error {
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
	_, resp, err := client.Repositories.UpdateBranchProtection(ctx, org, repoName, "master", &protectionRequest)
	if err != nil {
		log.Event(ctx, "update branch protection failed for master", log.Error(err))
	}
	_, resp, err = client.Repositories.UpdateBranchProtection(ctx, org, repoName, "develop", &protectionRequest)
	if err != nil {
		log.Event(ctx, "update branch protection failed for develop", log.Error(err))
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "master")
	if err != nil {
		log.Event(ctx, "adding protection, require signatures failed on branch master", log.Error(err), log.Data{"response": resp})
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "develop")
	if err != nil {
		log.Event(ctx, "adding protection, require signatures failed on branch develop", log.Error(err), log.Data{"response": resp})
	}
	return err
}

// setDevelopAsDefaultBranch will set the develop branch as the repositories default branch
func setDevelopAsDefaultBranch(ctx context.Context, client *github.Client, repoName string, repo *github.Repository) error {
	repo.AutoInit = nil
	_, resp, err := client.Repositories.Edit(ctx, org, repoName, repo)
	if err != nil {
		log.Event(ctx, "failed to set develop as the default branch", log.Error(err), log.Data{"response": resp})
	}
	return err
}

// createDevelopBranch will create a new branch named /develop, based on the last ref on master branch
func createDevelopBranch(ctx context.Context, client *github.Client, repoName string) error {
	ref, resp, err := client.Git.GetRef(ctx, org, repoName, "heads/master")
	if err != nil {
		log.Event(ctx, "get reference to master commit failed", log.Error(err), log.Data{"response": resp})
		return err
	}
	developBranch := "heads/develop"
	ref.Ref = &developBranch

	_, resp, err = client.Git.CreateRef(ctx, org, repoName, ref)
	if err != nil {
		log.Event(ctx, "create Reference to new develop branch failed", log.Error(err), log.Data{"response": resp})
		return err
	}
	return nil
}

// createRepo makes a call via the gitHubAPI to generate a new repository
func createRepo(ctx context.Context, client *github.Client, repo *github.Repository) error {
	newRepo, _, err := client.Repositories.Create(ctx, org, repo)
	fmt.Println(newRepo)
	if err != nil {
		log.Event(ctx, "repo creation failed", log.Error(err))
		return err
	}
	return nil
}

// getConfigurationsForNewRepo gets required configuration information from the end user
func getConfigurationsForNewRepo(name string, projType projectgeneration.ProjectType, personalAccessToken string, isLib bool) (accessToken, userHandle, repoName, repoDescription, defaultBranch string) {
	defaultBranch = "develop"
	if personalAccessToken == "" || personalAccessToken == "unset" {
		accessToken = promptForInput("Please provide your personal access token, to create one follow this guide https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line")
	}
	userHandle = promptForInput("Please provide your github handle/username")
	if name == "" || name == "unset" {
		repoName = promptForInput("Please provide the full name for the new repository (note 'unset' is not an applicable name')")
	} else {
		repoName = name
	}
	repoDescription = promptForInput("Please provide a description for the repository")
	if projType == "generic-project" || isLib {
		defaultBranch = "master"
	}
	return accessToken, userHandle, repoName, repoDescription, defaultBranch
}

// promptForInput gives a user a message and expect input to be provided
func promptForInput(prompt string) string {
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(prompt)
	scanner.Scan()
	input = scanner.Text()
	if scanner.Err() != nil {
		ctx := context.Background()
		log.Event(ctx, "repo creation failed", log.Error(scanner.Err()))
	}
	return input
}

// checkAccess ensures the user has provided a a valid access token
func checkAccess(ctx context.Context, client *github.Client) (bool, error) {
	_, _, err := client.Repositories.List(ctx, "", nil)
	if err != nil {
		log.Event(ctx, "failed to get list of repos", log.Error(err))
		return false, err
	}
	return true, nil
}
