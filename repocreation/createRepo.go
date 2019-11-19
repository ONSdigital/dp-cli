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

func GenerateGithubRepository() error {
	fmt.Println("\n\n\nThis script will create a new ONS Digital Publishing repository." +
		"In order to create and configure a new repository please answer the prompts.")

	accessToken, userHandle, repoName, repoDescription := getConfigurationsForNewRepo()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	org := "ONSdigital"

	checkAccess(ctx, client)
	log.Info("", log.Data{"repoDescription": repoDescription})
	// Team: "DigitalPublishing", slug: "digitalpublishing", id: 779417
	dpTeamSlug := "DigitalPublishing"
	teamID := int64(779417)

	repo := &github.Repository{
		Name:          github.String(repoName),
		Description:   github.String(repoDescription),
		DefaultBranch: github.String("develop"),
		MasterBranch:  github.String("master"),
		Private:       github.Bool(false),
		HasWiki:       github.Bool(false),
		HasProjects:   github.Bool(false),
		AutoInit:      github.Bool(true),
	}

	err := createRepo(client, ctx, org, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to create repository"), nil)
		return err
	}

	err = createDevelopBranch(client, ctx, org, repoName)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to create develop branch"), nil)
		return err
	}

	err = setDevelopAsDefaultBranch(client, ctx, org, repoName, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "failed to set default branch to develop"), nil)
	}

	err = setBranchProtections(client, ctx, org, repoName, dpTeamSlug)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to set all branch protections, please review and correct these manually"), nil)
	}

	setTeamsCollaborators(client, ctx, org, repoName, userHandle, teamID)
	return err
}

func setTeamsCollaborators(client *github.Client, ctx context.Context, org, repoName string, userHandle string, teamID int64) {
	addTeamRepoOptions := github.TeamAddTeamRepoOptions{Permission: "admin"}
	resp, err := client.Teams.AddTeamRepo(ctx, teamID, org, repoName, &addTeamRepoOptions)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to add collaborators"), log.Data{"resp": resp})
	}

	client.Repositories.RemoveCollaborator(ctx, org, repoName, userHandle)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Unable to remove self as a collaborator"), log.Data{"resp": resp})
	}
}

func setBranchProtections(client *github.Client, ctx context.Context, org, repoName, dpTeamSlug string) error {
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
		log.ErrorCtx(ctx, errors.Wrap(err, "Update branch protection failed for master"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.UpdateBranchProtection(ctx, org, repoName, "develop", &protectionRequest)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Update branch protection failed for develop"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "master")
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Adding protection, require signatures failed on branch master"), log.Data{"resp": resp})
	}
	_, resp, err = client.Repositories.RequireSignaturesOnProtectedBranch(ctx, org, repoName, "develop")
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Adding protection, require signatures failed on branch develop"), log.Data{"resp": resp})
	}
	return err
}
func setDevelopAsDefaultBranch(client *github.Client, ctx context.Context, org, repoName string, repo *github.Repository) error {
	repo.AutoInit = nil
	_,resp,err := client.Repositories.Edit(ctx,org,repoName, repo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Failed to set develop as the default branch"), log.Data{"resp": resp})
	}
	return err
}

func createDevelopBranch(client *github.Client, ctx context.Context, org, repoName string) error {
	ref, resp, err := client.Git.GetRef(ctx, org, repoName, "heads/master")
	developBranch := "heads/develop"
	ref.Ref = &developBranch
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Get Reference to master commit failed"), log.Data{"resp": resp})
		return err
	}
	_, resp, err = client.Git.CreateRef(ctx, org, repoName, ref)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "Create Reference to new develop branch failed"), log.Data{"resp": resp})
		return err
	}
	return nil
}

func createRepo(client *github.Client, ctx context.Context, org string, repo *github.Repository) error {
	newRepo, _, err := client.Repositories.Create(ctx, org, repo)
	fmt.Println(newRepo)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "repo creation failed"), nil)
		return err
	}
	return nil
}

func getConfigurationsForNewRepo() (accessToken, userHandle, repoName, repoDescription string) {
	accessToken = promptForInput("Please provide your user access token, to create one follow this guide https://help.github.com/en/github/authenticating-to-github/creating-a-personal-access-token-for-the-command-line")
	userHandle = promptForInput("Please provide your github handle/username")
	repoName = promptForInput("Please provide the full name for the new repository")
	repoDescription = promptForInput("Please provide a description for the repository")
	return accessToken, userHandle, repoName, repoDescription
}

func promptForInput(prompt string) string {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSuffix(input, "\n")
}

func checkAccess(ctx context.Context, client *github.Client) {
	_, _, err := client.Repositories.List(ctx, "", nil)
	if err != nil {
		log.ErrorCtx(ctx, errors.Wrap(err, "failed to get list of repos"), nil)
		os.Exit(3)
	}
}
