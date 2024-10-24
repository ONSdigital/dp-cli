# Repo Generation Tool

## WARNING

Interim actions required after completion of generating a repository

- Change branch name `main` to `master`
- Update security settings for the `master` branch to match that of `main`

For exact instructions on how to complete this please read the following guide [MAIN_TO_MASTER](MAIN_TO_MASTER_GUIDE.md).
These steps should be removed once CI has been updated to use `main` as the leading branch.

### Usage prerequisite

#### GitHub 'Personal Access Token'

1. Navigate to github.com
2. Click your Avatar
3. Go to Settings
4. Go to Developer Settings
5. Click Personal Access Token
6. Generate an Access token with permissions to manipulate repositories

#### Be a member of the ONS Dissemination team and the ONSDigital organisation with permissions to create new repositories

___

### How to use

1. (optional) Export the environment variable `GITHUB_PERSONAL_ACCESS_TOKEN` with your github personal access token
2. Run `dp create-repo github`
3. Enter your 'Personal Access Token' when prompted (if step 1. was not complete)
4. Enter your GitHub handle when prompted
5. Enter your repository name, note that if the repository is specific to the Dissemination Platform then prefix it with `dis-`
6. Enter a description for the repository when prompted
