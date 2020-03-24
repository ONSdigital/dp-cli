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
dp-setup-path: "path/to/your/dp-setup/project" # The path to the dp-setup repo on your machine.
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
ssh-user: JamesHetfield
environments:
  - name: production
    profile: production
  - name: develop
    profile: development
  - name: cmd-dev
    profile: development
```

Create an environment variable `DP_CLI_CONFIG` assigning the path to config file you just created.

Example:
```
export DP_CLI_CONFIG="<YOUR_PATH>/dp-cli-config.yml"
```

### Build and run

Build, install and start the cli:
```
make install
dp
```
Or to build a binary locally:
```
make build
./dp
```

You should be presented you a help menu similar to:
```bash
dp is a command line client providing handy helper tools for ONS Digital Publishing software engineers

Usage:
  dp [command]

Available Commands:
  clean            Delete data from your local environment
  create-repo      Creates a new repository with the typical Digital Publishing configurations
  generate-project Generates the boilerplate for a given project type
  help             Help about any command
  import           Import data into your local developer environment
  remote           Allow or deny remote access to environment
  spew             Log out some useful debugging info
  ssh              Access an environment using ssh
  version          Print the app version

Flags:
  -h, --help   help for dp

Use "dp [command] --help" for more information about a command.
```

Use the available commands for more info on the functionality available.

### Common issues

#### Credentials error

`error creating group commands for env: develop: error fetching ec2: {"develop" "development"}: NoCredentialProviders: no valid providers in chain. Deprecated.`

Ensure you have AWS credentials set up as documented here: https://github.com/ONSdigital/dp-setup/blob/develop/AWS-CREDENTIALS.md

If you do not want to set up separate profiles, another option is to not specify any profiles in your `dp-cli-config.yml`. That way the default credentials will be used.

```
environments:
  - name: production
    profile:
  - name: develop
    profile:
  - name: cmd-dev
    profile:
```

#### SSH command fails

```
➜  dp ssh develop
ssh to develop
```

If the SSH command fails, ensure that the `dp remote allow` command has been run for the environment you want to SSH into.

#### Remote Allow security group error

`Error: no security groups matching environment: "develop" with name "develop - bastion"`

Ensure you have `region=eu-west-1` in your AWS configuration.

Depending on the command you're trying to run, and what you're trying to access, ensure your `AWS_DEFAULT_PROFILE` is set correctly.

#### Remote Allow security group rule already exists error

```
➜  dp remote allow develop
[dp] allowing access to develop
Error: error adding rules to bastionSG: InvalidPermission.Duplicate: the specified rule "peer: X.X.X.X/32, TCP, from port: 22, to port: 22, ALLOW" already exists
        status code: 400, request id: 26a61345-8391-4c65-bfd7-4f0052892b6b
```

The error occurs when rules have previously been added and the command is run again. Use `dp remote deny` to clear out existing rules and try again.
