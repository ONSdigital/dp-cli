# dp-cli

:warning: Still in active development. If you noticed and bugs/issues please open a Github issue. 

## Prerequisites
`dp-cli` uses Go Modules so requires a go version of **1.11** or later. 

`dp-cli` requires:
- `dp-code-list-scripts` 
- `dp-hierarchy-builder`

to be one your `$GOPATH`:
```shell script
go get github.com/ONSdigital/dp-code-list-scripts
go get github.com/ONSdigital/dp-hierarchy-builder
```

### Getting started
Clone the code
```shell script
git clone git@github.com:ONSdigital/dp-cli.git
```
:warning: `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

### Config
Add the following to your bash profile - replacing `<PATH_TO_PROJECT>` with the appropriate path for your set up. 

```shell script
export DP_CLI_CFG="<PATH_TO_PROJECT>/dp-cli/config/config.yml"
```
Build and run the binary
````shell script
go build -o dp-cli
./dp-cli
````

You should be presented you a help menu similar to:
```shell script
dp-cli provides util functions for developers in ONS Digital Publishing

Usage:
  dp-cli [command]

Available Commands:
  clean            Clean/Delete data from your local environment
  create-repo      Creates a new repository with the typical Digital Publishing configurations
  generate-project Generates the boilerplate for a given project type
  help             Help about any command
  import           ImportData your local developer environment
  version          Print the app version

Flags:
  -h, --help   help for dp-cli

Use "dp-cli [command] --help" for more information about a command.
```

#### Clean out all CMD data from you local env:
```shell script
./dp-cli clean cmd
```

#### Import the generic hierarchy and suicides code list:
```shell script
./dp-cli import cmd
```

#### Create a repository on github
```shell script
./dp-cli create-repo github
```
Note: further details found at [README.md](repository-creation/README.md)
#### Generate a project
```shell script
./dp-cli generate-project
```
Note: further details found at [README.md](project-generation/README.md)
#### Generate a project and host the repository on github
```shell script
./dp-cli generate-project --create-repo yes
```
Note: further details found at [COMPLETE_PROJECT_SETUP.md](project-generation/COMPLETE_PROJECT_SETUP.md)