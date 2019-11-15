# dp-utils

:warning: Still in active development. If you noticed and bugs/issues please open a Github issue. 

## Prerequisites
`dp-utils` uses Go Modules so requires a go version of **1.11** or later. 

`dp-utils` requires:
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
git clone git@github.com:ONSdigital/dp-utils.git
```
:warning: `dp-utils` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

### Config
Add the following to your bash profile - replacing `<PATH_TO_PROJECT>` with the appropriate path for your set up. 

```
export DP_UTILS_CFG="<PATH_TO_PROJECT>/dp-utils/config/config.yml"
```
Build and run the binary
````bash
go build -o dp-utils
./dp-utils
````

You should be presented you a help menu similar to:
```
dp-utils provides util functions for developers in ONS Digital Publishing

Usage:
  dp-utils [command]

Available Commands:
  clean       Clean/Delete data from your local environment
  help        Help about any command
  import      ImportData your local developer environment
  version     Print the app version

Flags:
  -h, --help   help for dp-utils

Use "dp-utils [command] --help" for more information about a command.
```

Clean out all CMD data from you local env:
```
./dp-utils clean cmd
```

Import the generic hierarchy and suicides code list:
```
./dp-utils import cmd
```