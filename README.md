# dp-cli

:warning: Still in active development. If you noticed and bugs/issues please open a Github issue. 

## Prerequisites
`dp-cli` uses Go Modules so requires a go version of **1.11** or later. 

`dp-cli` requires:
- `dp-code-list-scripts` 
- `dp-hierarchy-builder`

to be one your `$GOPATH`:
```
go get github.com/ONSdigital/dp-code-list-scripts
go get github.com/ONSdigital/dp-hierarchy-builder
```

### Getting started
Clone the code
```
git clone git@github.com:ONSdigital/dp-cli.git
```

:warning: `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

### Config
Add the following to your bash profile - replacing `<PATH_TO_PROJECT>` with the appropriate path for your set up. 

```
export DP_CLI_CONFIG="<PATH_TO_PROJECT>/dp-cli/config/config.yml"
```
Build and run the binary
```
go build -o dp-cli
./dp-cli
```

You should be presented you a help menu similar to:
```
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
```
./dp-cli clean cmd
```

#### Create a repository on github
```
./dp-cli create-repo github
```
