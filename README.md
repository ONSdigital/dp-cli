# dp-utils

:warning: Still in active development. If you noticed and bugs/issues please open a Github issue. 

## Prerequisites
`dp-utils` uses Go Modules so requires a go version of **1.11** or later. 

`dp-utils` requires:
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
git clone git@github.com:ONSdigital/dp-utils.git
```
:warning: `dp-utils` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

### Config
Add the following to your bash profile - replacing `<PATH_TO_PROJECT>` with the appropriate path for your set up. 

```shell script
export DP_UTILS_CFG="<PATH_TO_PROJECT>/dp-utils/config/config.yml"
```
Build and run the binary
````shell script
go build -o dp-utils
./dp-utils
````

You should be presented you a help menu similar to:
```shell script
dp-utils provides util functions for developers in ONS Digital Publishing

Usage:
  dp-utils [command]

Available Commands:
  clean            Clean/Delete data from your local environment
  create-repo      Creates a new repository with the typical Digital Publishing configurations
  generate-project Generates the boilerplate for a given project type
  help             Help about any command
  import           ImportData your local developer environment
  version          Print the app version

Flags:
  -h, --help   help for dp-utils

Use "dp-utils [command] --help" for more information about a command.
```

#### Clean out all CMD data from you local env:
```shell script
./dp-utils clean cmd
```

#### Import the generic hierarchy and suicides code list:
```shell script
./dp-utils import cmd
```

#### Create a repository on github
// TODO prerequisite - user access token
```shell script
./dp-utils create-repo github
```

#### Generate a project
```shell script
./dp-utils generate-project
```
##### Optional flags
--name :              The name of the application, if Digital specific application it should be prepended with 'dp-'

--go-version :        The version of Go the application should use (Not used on generic-programs)

--project-location :  Location to generate project in

--create-repository : Should a repository be created for the project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")

--type :              Type of application to generate, values can be: 'generic-program', 'base-application', 'api', 'controller', 'event-driven'")
