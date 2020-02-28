# dp-cli

#### Command line client providing *handy helper tools* for the ONS Digital Publishing software engineering team.

:warning: Still in active development. If you noticed and bugs/issues please open a Github issue. 

### Getting started
Clone the code
```
git clone git@github.com:ONSdigital/dp-cli.git
```

:warning: `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

#### Prerequisites
`dp-cli` uses Go Modules so requires a go version of **1.11** or later. 

`dp-cli` requires:
- `dp-code-list-scripts` 
- `dp-hierarchy-builder`

to be on your `$GOPATH`:
```
go get github.com/ONSdigital/dp-code-list-scripts
go get github.com/ONSdigital/dp-hierarchy-builder
```

`dp-cli` also depends on `dp-setup` for  environment config:
```
git clone git@github.com:ONSdigital/dp-setup.git
```


### Configuration
`dp-cli` configuration is defined in a `.yml` configuration file and the cli expects an environment variable providing the config file path.

Create a new `dp-cli-config.yml` file and add the example content below (update as required to match your local set up):

```yaml
## Example config file Replace fields as required
cmd:
  neo4j-url: bolt://localhost:7687
  mongo-url: localhost:27017
  mongo-dbs:  # The mongo databases to be dropped when cleaning your CMD data
    - "imports"
    - "datasets"
    - "filters"
    - "codelists"
    - "test"
  hierarchies: # The hierarchies import scripts to run when importing CMD data.
    - "admin-geography.cypher"
    - "cpih1dim1aggid.cypher"

  codelists: # The CMD codelist import scripts to run when importing CMD data.
    - "opss.yaml"
ssh:
  environments:
    - name: production
      profile: production
    - name: develop
      profile: development
    - name: cmd
      profile: development
dp-setup-path: "path/to/your/dp-setup/project" # The path to the dp-setup repo on your machine.
```
 
Create an environment variable `DP_CLI_CONFIG` assigning the path to config file you just created.

Example:
```
export DP_CLI_CONFIG="<YOUR_PATH>/dp-ci-config.yml"
```

### Build and run

Build, install and start the cli:
```
make install
dp-cli
```
Or to build a binary locally:
```
make build
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
  spew             log out some useful debugging info
  ssh              access an environment using ssh
  version          Print the app version

Flags:
  -h, --help   help for dp-cli

Use "dp-cli [command] --help" for more information about a command.
```

Use the available commands for more info on the functionality available.