# dp-cli

Command-line client providing *handy helper tools* for the ONS Digital Publishing software engineering team

:warning: Still in active development. If you notice any bugs/issues please open a Github issue.

### Getting started

Clone the code

```sh
git clone git@github.com:ONSdigital/dp-cli.git
```

:warning: `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

#### Prerequisites

`dp-cli` uses Go Modules so requires a go version of **1.11** or later.

`dp-cli` requires (for some functionality) the repos:

- `dp-code-list-scripts`
- `dp-hierarchy-builder`

to be on your `$GOPATH`:

```sh
go get github.com/ONSdigital/dp-code-list-scripts
go get github.com/ONSdigital/dp-hierarchy-builder
```

`dp-cli` also depends on `dp-setup` for  environment config:

```sh
git clone git@github.com:ONSdigital/dp-setup.git
```

### Configuration

`dp-cli` configuration is defined in a YAML file and the CLI expects an environment variable `DP_CLI_CONFIG` with the path to that config file.

Create a new `dp-cli-config.yml` file and add the example content below (edit to match your local setup):

```yaml
## Example config file - replace fields as required
dp-setup-path: "/path/to/your/clone/of/dp-setup" # The path to the dp-setup repo on your machine
cmd:
  neo4j-url: bolt://localhost:7687
  mongo-url: localhost:27017
  mongo-dbs:  # The mongo databases to be dropped when cleaning your data
    - "imports"
    - "datasets"
    - "filters"
    - "codelists"
    - "test"
  hierarchies: # The hierarchies import scripts to run when importing data
    - "admin-geography.cypher"
    - "cpih1dim1aggid.cypher"

  codelists: # The codelist import scripts to run when importing data
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

Create an environment variable `DP_CLI_CONFIG` with the path to the above file:

```sh
export DP_CLI_CONFIG="<YOUR_PATH>/dp-cli-config.yml"
```

### Build and run

Build, install and start the CLI:

```sh
make install
dp
```

Or to build a binary locally:

```sh
make build
./dp
```

You should be presented with a help menu similar to:

```text
dp is a command-line client providing handy helper tools for ONS Digital Publishing software engineers

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

[Ensure you have AWS credentials set up](https://github.com/ONSdigital/dp/blob/master/guides/AWS_CREDENTIALS.md).

If you do not want to set up separate profiles, another option is to not specify any profiles in your `dp-cli-config.yml`. That way the default credentials will be used.

```yaml
environments:
  - name: production
    profile:
  - name: develop
    profile:
  - name: cmd-dev
    profile:
```

#### SSH command fails

```sh
➜  dp ssh develop
ssh to develop
```

If the SSH command fails, ensure that the `dp remote allow` command has been run for the environment you want to SSH into.

#### Remote Allow security group error

`Error: no security groups matching environment: "develop" with name "develop - bastion"`

Ensure you have `region=eu-west-1` in your AWS configuration.

Depending on the command you're trying to run, and what you're trying to access, ensure your `AWS_DEFAULT_PROFILE` is set correctly.

#### Remote Allow security group rule already exists error

```sh
$ dp remote allow develop
[dp] allowing access to develop
Error: error adding rules to bastionSG: InvalidPermission.Duplicate: the specified rule "peer: X.X.X.X/32, TCP, from port: 22, to port: 22, ALLOW" already exists
        status code: 400, request id: 26a61345-8391-4c65-bfd7-4f0052892b6b
```

The error occurs when rules have previously been added and the command is run again.
Use `dp remote deny $env` to clear out existing rules and try again.

This error should no longer appear - the code should now avoid re-adding existing rules.
However, it is possible that the rule has been added with a description that does not match your username.
If so, you will have to use the AWS web UI/console to remove any offending security group rules.

### Advanced use

#### Manually configuring your IP

Optionally, (e.g. to avoid the program looking up your IP), you can use an environment variable `MY_IP` to force the IP used when running `dp remote allow`, for example:

```sh
MY_IP=192.168.11.22 dp remote allow develop
```
