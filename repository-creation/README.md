## Repo Generation Tool

### Usage prerequisite 

#### Github 'Personal Access Token'
    1. Navigate to github.com
    2. Click your Avatar
    3. Go to Settings
    4. Go to Developer Settings
    5. Click Personal Access Token
    6. Generate an Access token with permissions to manipulate repositories
    
#### Be a member of the ONS Digital Publishing team and the ONSDigital organisation with permissions to create new repositories
___
### How to use
1. (optional) Export the environment variable GITHUB_PERSONAL_ACCESS_TOKEN with your github personal access token
2. Run ./dp-cli repo-generation github
3. Enter your 'Personal Access Token' when prompted (if step 1. was not complete)
4. Enter your Github handler when prompted
5. Enter your repository name, note that if the repository is specific to Digital Publishing then prefix it with 'dp-'
6. Enter a description for the repository when prompted