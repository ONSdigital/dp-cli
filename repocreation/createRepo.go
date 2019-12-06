package repocreation

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ONSdigital/go-ns/log"
	"github.com/google/go-github/v28/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"os"
	"strings"
)

const (
	org = "ONSdigital"
	// Team: "DigitalPublishing", slug: "digitalpublishing", id: 779417
	dpTeamSlug    = "DigitalPublishing"
	teamID        = int64(779417)
	defaultBranch = "develop"
	masterBranch  = "master"
)

// GenerateGithubRepository is the entry point to generating the repository
func GenerateGithubRepository(name string) error {
	fmt.Println("This script will create a new ONS Digital Publishing repository." +
		"In order to create and configure a new repository please answer the prompts.")

	accessToken, userHandle, repoName, repoDescription := getConfigurationsForNewRepo(name)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	checkAccess(ctx, client)

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

	err := createRepo(client, ctx, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "unable to create repository"), nil)
		return err
	}

	err = createDevelopBranch(client, ctx, repoName)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "unable to create develop branch"), nil)
		return err
	}

	err = setDevelopAsDefaultBranch(client, ctx, repoName, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "failed to set default branch to develop"), nil)
		return err
	}

	err = setBranchProtections(client, ctx, repoName)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "unable to set all branch protections, please review and correct these manually"), nil)
		return err
	}

	err = setTeamsAndCollaborators(client, ctx, repoName, userHandle)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to set team and collaborators"), nil)
		return err
	}

	// Notify user of completion and get them to turn off actions
	log.Info("repository has successfully been create please disable actions for this repository", nil)
	return nil
}

// setTeamsAndCollaborators will set the DigitalPublishing team as a team working on the repo and removes the creator from being a collaborator
func setTeamsAndCollaborators(client *github.Client, ctx context.Context, repoName string, userHandle string) error {
	addTeamRepoOptions := github.TeamAddTeamRepoOptions{Permission: "admin"}
	resp, err := client.Teams.AddTeamRepo(ctx, teamID, org, repoName, &addTeamRepoOptions)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "unable to add collaborators"), log.Data{"resp": resp})
	}

	resp, err = client.Repositories.RemoveCollaborator(ctx, org, repoName, userHandle)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "unable to remove self as a collaborator"), log.Data{"resp": resp})
	}
	return err
}

// setBranchProtections sets the protections for both master and develop branches
func setBranchProtections(client *github.Client, ctx context.Context, repoName string) error {
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
		log.ErrorCtx(ctx, errors.Wrap(err, "update branch protection failed for master"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.UpdateBranchProtection(ctx, org, repoName, "develop", &protectionRequest)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "update branch protection failed for develop"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "master")
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "adding protection, require signatures failed on branch master"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "develop")
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "adding protection, require signatures failed on branch develop"), log.Data{"resp": resp})
	}
	return err
}

// setDevelopAsDefaultBranch will set the develop branch as the repositories default branch
func setDevelopAsDefaultBranch(client *github.Client, ctx context.Context, repoName string, repo *github.Repository) error {
	repo.AutoInit = nil
	_, resp, err := client.Repositories.Edit(ctx, org, repoName, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "failed to set develop as the default branch"), log.Data{"resp": resp})
	}
	return err
}

// createDevelopBranch will create a new branch named /develop, based on the last ref on master branch
func createDevelopBranch(client *github.Client, ctx context.Context, repoName string) error {
	ref, resp, err := client.Git.GetRef(ctx, org, repoName, "heads/master")
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "get reference to master commit failed"), log.Data{"resp": resp})
		return err
	}
	developBranch := "heads/develop"
	ref.Ref = &developBranch

	_, resp, err = client.Git.CreateRef(ctx, org, repoName, ref)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "create reference to new develop branch failed"), log.Data{"resp": resp})
		return err
	}
	return nil
}

// createRepo makes a call via the gitHubAPI to generate a new repository
func createRepo(client *github.Client, ctx context.Context, repo *github.Repository) error {
	newRepo, _, err := client.Repositories.Create(ctx, org, repo)
	fmt.Println(newRepo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "repo creation failed"), nil)
		return err
	}
	return nil
}

// getConfigurationsForNewRepo gets required configuration information from the end user
func getConfigurationsForNewRepo(name string) (accessToken, userHandle, repoName, repoDescription string) {
	accessToken = promptForInput("Please provide your user access token, to create one follow this guide https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line")
	userHandle = promptForInput("Please provide your github handle/username")
	if name == "dp-unnamed-application" || len(name) < 1 {
		repoName = promptForInput("Please provide the full name for the new repository")
	}
	repoDescription = promptForInput("Please provide a description for the repository")
	return accessToken, userHandle, repoName, repoDescription
}

// promptForInput gives a user a message and expect input to be provided
func promptForInput(prompt string) string {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSuffix(input, "\n")
}

// checkAccess ensures the user has provided a a valid access token
func checkAccess(ctx context.Context, client *github.Client) {
	_, _, err := client.Repositories.List(ctx, "", nil)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "failed to get list of repos"), nil)
		os.Exit(3)
	}
}
