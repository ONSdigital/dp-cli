package repository_creation

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ONSdigital/dp-cli/project_generation"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/google/go-github/v66/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	org        = "ONSdigital"
	mainBranch = "main"
)

type GitHubTeam struct {
	slug string
	name string
}

// List of standard GitHub Teams to apply permissions
var (
	disseminationTeam = GitHubTeam{
		slug: "dissemination",
		name: "Dissemination",
	}
	disseminationTechLeadTeam = GitHubTeam{
		slug: "dissemination-tech-leads",
		name: "Dissemination Tech Leads",
	}
)

func RunGenerateRepo(cmd *cobra.Command, args []string) error {
	var err error
	nameOfApp, _ := cmd.Flags().GetString("name")
	token, _ := cmd.Flags().GetString("token")
	branchStrategyInput, _ := cmd.Flags().GetString("strategy")
	branchStrategy := strings.ToLower(strings.TrimSpace(branchStrategyInput))
	teamSlugsInput, _ := cmd.Flags().GetString("team-slugs")
	_, err = GenerateGithub(nameOfApp, "", "", token, branchStrategy, teamSlugsInput)
	if err != nil {
		return err
	}

	return nil
}

// GenerateGithub is the entry point to generating the repository
func GenerateGithub(name, description string, ProjectType project_generation.ProjectType, personalAccessToken, branchStrategy string, teamSlugsInput string) (cloneUrl string, err error) {
	accessToken, repoName, repoDescription, defaultBranch, repoTeamSlugsInput := getConfigurationsForNewRepo(name, description, ProjectType, personalAccessToken, branchStrategy, teamSlugsInput)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	teamSlugs := strings.Split(repoTeamSlugsInput, ",")

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
	log.Info(ctx, "repo", log.Data{
		"repo": repo,
	})

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

	err = setTeamsAndCollaborators(ctx, client, repoName, teamSlugs)
	if err != nil {
		log.Error(ctx, "unable to set team and collaborators", err)
		return cloneUrl, err
	}

	err = setLabels(ctx, client, repoName)
	if err != nil {
		log.Error(ctx, "unable to set labels", err)
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
func setTeamsAndCollaborators(ctx context.Context, client *github.Client, repoName string, teamSlugs []string) error {
	// Add Diss as write
	_, err := addTeamToRepo(ctx, client, disseminationTeam.slug, "push", org, repoName)
	if err != nil {
		log.Error(ctx, "unable to add dissemination collaborators", err)
		return err
	}

	// Add TLs as admins
	_, err = addTeamToRepo(ctx, client, disseminationTechLeadTeam.slug, "admin", org, repoName)
	if err != nil {
		log.Error(ctx, "unable to add technical lead collaborators", err)
		return err
	}

	// Add teams as maintainers
	for _, slug := range teamSlugs {
		if slug != "" {
			_, err = addTeamToRepo(ctx, client, slug, "maintain", org, repoName)
			if err != nil {
				log.Error(ctx, "unable to add team collaborators", err)
				return err
			}
		}
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

func addTeamToRepo(ctx context.Context, client *github.Client, slug, permission, org, repoName string) (*github.Response, error) {
	addTeamRepoOptions := github.TeamAddTeamRepoOptions{Permission: permission}
	return client.Teams.AddTeamRepoBySlug(ctx, org, slug, org, repoName, &addTeamRepoOptions)
}

// setBranchProtections sets the protections for both main and develop branches
func setBranchProtections(ctx context.Context, client *github.Client, repoName, branchStrategy string) error {
	requiredStatusChecks := github.RequiredStatusChecks{
		Strict:   true,
		Contexts: &[]string{},
	}

	dismissalRestrictionsRequest := github.DismissalRestrictionsRequest{
		Users: &[]string{},
		Teams: &[]string{disseminationTeam.slug},
		Apps:  &[]string{},
	}
	requiredPullRequestReviewsEnforcementRequest := github.PullRequestReviewsEnforcementRequest{
		DismissalRestrictionsRequest: &dismissalRestrictionsRequest,
		DismissStaleReviews:          true,
		RequiredApprovingReviewCount: 1,
	}

	branchRestrictions := github.BranchRestrictionsRequest{
		Users: []string{},
		Teams: []string{disseminationTeam.slug},
		Apps:  []string{},
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
func getConfigurationsForNewRepo(name, description string, projType project_generation.ProjectType, personalAccessToken, branchStrategy, teamSlugsInput string) (accessToken, repoName, repoDescription, defaultBranch, repoTeamSlugsInput string) {
	ctx := context.Background()

	defaultBranch = "develop"
	if personalAccessToken == "" {
		token, exists := os.LookupEnv("GITHUB_PERSONAL_ACCESS_TOKEN")
		if exists {
			accessToken = token
		} else {
			accessToken = PromptForInput("Please provide your personal access token (to create one follow this guide https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens):")
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
		branchStrategy, err := project_generation.OptionPromptInput(ctx, prompt, options...)
		if err != nil {
			log.Error(ctx, "error getting branch strategy", err)
		}
		branchStrategy = strings.Replace(branchStrategy, " flow", "", -1)
	}
	if projType == "generic-project" || branchStrategy == "github" {
		defaultBranch = "main"
	}
	if teamSlugsInput == "" {
		prompt := "Please set at least one team in a comma separated list who will be responsible for this repo."
		teamSlugsInput = PromptForInput(prompt)
	}
	return accessToken, repoName, repoDescription, defaultBranch, teamSlugsInput
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

// removeDefaultLabels deletes all default labels that are made by default when a repo in generated
func removeDefaultLabels(ctx context.Context, client *github.Client, repoName string) error {
	options := github.ListOptions{}
	labels, resp, err := client.Issues.ListLabels(ctx, org, repoName, &options)
	if err != nil {
		log.Error(ctx, "unable to get all labels", err, log.Data{"response": resp})
		return err
	}
	for _, label := range labels {
		resp, err := client.Issues.DeleteLabel(ctx, org, repoName, *label.Name)
		if err != nil {
			log.Error(ctx, "unable to delete label: "+*label.Name, err, log.Data{"response": resp})
			return err
		}
	}
	return nil
}

// setLabels adds all labels generated by generateLabels to the repo being generated and removes all default labels
func setLabels(ctx context.Context, client *github.Client, repoName string) error {
	removeDefaultLabels(ctx, client, repoName)
	labels := generateLabels()
	for _, label := range labels {
		_, resp, err := client.Issues.CreateLabel(ctx, org, repoName, label)
		if err != nil {
			log.Error(ctx, "unable to create label for "+*label.Name, err, log.Data{"response": resp})
			return err
		}
	}
	return nil
}

func generateLabels() []*github.Label {
	alpha := github.Label{
		Name:        pointer("alpha"),
		Color:       pointer("87115b"),
		Description: pointer("Use when pre-releasing a new version that is unstable due to knowledge of further breaking changes"),
	}
	beta := github.Label{
		Name:        pointer("beta"),
		Color:       pointer("206095"),
		Description: pointer("Use when pre-releasing a new version that may have future breaking changes but these are unknown"),
	}
	breaking := github.Label{
		Name:        pointer("breaking"),
		Color:       pointer("003C57"),
		Description: pointer("Use when releasing a new version that is stable, it doesn’t have to be backward compatible"),
	}
	feature := github.Label{
		Name:        pointer("feature"),
		Color:       pointer("0f8243"),
		Description: pointer("Use when minor changes are made such as a new feature that doesn’t break backward compatibility"),
	}
	bug := github.Label{
		Name:        pointer("bug"),
		Color:       pointer("d0021b"),
		Description: pointer("Use when there is a feature to be fixed, it must be backward compatible"),
	}
	maintenance := github.Label{
		Name:        pointer("maintenance"),
		Color:       pointer("746cb1"),
		Description: pointer("Use when there is a security or performance bug to be fixed, it must be backward compatible"),
	}

	labels := []*github.Label{
		&alpha,
		&beta,
		&breaking,
		&feature,
		&bug,
		&maintenance,
	}
	return labels
}

// create a pointer to a string
func pointer(name string) (pointerToString *string) {
	pointerToString = &name
	return
}
