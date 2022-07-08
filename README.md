# dp-cli

Command-line client providing *handy helper tools* for the ONS Digital Publishing software engineering team

:warning: Still in active development. If you notice any bugs/issues please open a Github issue.

## Getting started

Clone the code (not needed if [installing on macOS](#brew-installation))

```sh
git clone git@github.com:ONSdigital/dp-cli.git
```

:warning: `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

### Prerequisites

**Required:**

The DP CLI uses Go Modules so requires a go version of **1.18** or later.

Check you have a suitable version of `go` installed with:

`go version`

(Ideally 1.18)

[ The following will ensure version 1.18

  ```sh
  brew install go@1.18
  brew unlink go
  brew link —force go@1.18
  ```

Check desired version of `go` is on your PATH with `echo $PATH` and if not, either edit your .zshrc file to have the correct path OR do:

  ```sh
  echo 'export GOPATH=$HOME/go' >> ~/.zshrc
  echo 'export PATH="/usr/local/opt/go@1.18/bin:$PATH"' >> ~/.zshrc
  ```

  and restart the terminal ]

Ensure `session-manager-plugin` is installed by running the following command
```
 which session-manager-plugin
 ```
if not installed, follow this [doc](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html#install-plugin-macos)


**Optional:**

 The following are only required for some functionality of this tool.

In order to use the `dp ssh` sub-command you will need:

- [`dp-setup`](https://github.com/ONSdigital/dp-setup) cloned locally:

  ```bash
  git clone git@github.com:ONSdigital/dp-setup
  ```
- [`dp-ci`](https://github.com/ONSdigital/dp-ci) cloned locally:

  ```bash
  git clone git@github.com:ONSdiqgital/dp-ci
  ```
Note: Make sure `dp-setup` and `dp-ci` branch points to main locally. This is necessary as it has the required SSH configuration and the relavant inventories.

In order to use the `dp import cmd` sub-command (e.g. when you are using Neo4j; `import` is currently *not needed* if you are using Neptune) you will need:

- [`dp-code-list-scripts`](https://github.com/ONSdigital/dp-code-list-scripts) cloned locally:

  ```bash
  git clone git@github.com:ONSdigital/dp-code-list-scripts
  ```

- [`dp-hierarchy-builder`](https://github.com/ONSdigital/dp-hierarchy-builder) cloned locally:

  ```bash
  git clone git@github.com:ONSdigital/dp-hierarchy-builder
  ```

### Configuration

Configuration is defined in a YAML file. By default the CLI expects the config to be in `~/.dp-cli-config.yml`. The config file location can be customised by setting `DP_CLI_CONFIG` environment variable to your chosen path.

The [sample config file](./config/example_config.yml) should be tailored to suit you. For example:

```bash
cp -i config/example_config.yml ~/.dp-cli-config.yml
vi ~/.dp-cli-config.yml
```

update the paths and ssh-user: 

```yaml
    dp-setup-path: path to your local dp-setup
    dp-ci-path: path to your local dp-ci
    dp-hierarchy-builder-path: path to your local dp-hierarchy-builder-path
    dp-code-list-scripts-path: path to your local dp-code-list-scripts-path
    ssh-user: Your first and last name concatenated eg. JaneBloggs"
```

and if this is a first time setup, comment out `prod` from environments, thus:

```yaml
     #- name: prod 
     #  profile: dp-prod
     #  user: ubuntu 
     #  tag: awsb
```

*Note*: **ssh-user** is a string used to put your name against SecurityGroup changes.

### Brew Installation

If using macOS, you can now install using `brew`:

- Create tap

   ```sh
   brew tap ONSdigital/homebrew-dp-cli git@github.com:ONSdigital/homebrew-dp-cli
   ```

- Run brew install

   ```sh
   brew install dp-cli
   ```

### Build and run

Build, install and start the CLI:

```sh
make install
dp
```

[
  If you get:

  `zsh: command not found: dp`

Then either edit your .zshrc file have the correct path OR do:

```sh
echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.zshrc
```

  and restart the terminal ]

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
  scp              Push (or `--pull`) a file to (from) an environment using scp
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

1. If sandbox/prod/staging are not in the dp cli output try unsetting `AWS_REGION` and `AWS_DEFAULT_REGION`

1. `SSOProviderInvalidToken: the SSO session has expired or is invalid`

    If you see the above error, you need to re-authenticate with sign in information

1. `error fetching ec2: {Name:sandbox Profile:dp-sandbox User:ubuntu Tag:awsb CI:false ExtraPorts:{Bastion:[] Publishing:[] Web:[]}}: MissingRegion: could not find region configuration`

    check that you have the correct AWS profile names in your `~/.aws/config` file (dp-sandbox, dp-staging, dp-prod, dp-ci). A sample config for `~/.aws/config` is included at the end of this guide as a reference.

1. `Error: no security groups matching environment: "sandbox" with name "sandbox - bastion"`

    check  ~/.aws/credentials and remove any profile information added for dp-sandbox, dp-staging and dp-prod as this is not needed

    If you do not want to set up separate profiles, another option is to not specify any profiles in your `~/.dp-cli-config.yml`. That way the default credentials will be used.

    ```yaml
    environments:
      - name: prod
        profile:
        user: ubuntu 
        tag: awsb
      - name: staging
        profile:
        user: ubuntu 
        tag: awsb
    ```

#### SSH/SCP command fails

```sh
$ dp ssh develop
ssh to develop
```

If the SSH or SCP command fails, ensure that the `dp remote allow` command has been run for the environment you want to connect to.

#### Remote Allow security group error

`Error: no security groups matching environment: "sandbox" with name "sandbox - bastion"`

Ensure you have `region=eu-west-2` in your AWS configuration.

Depending on the command you're trying to run, and what you're trying to access, ensure your `AWS_PROFILE` is set correctly and there is no prod/sandbox/ci config added in the `~/.aws/credentials` file.
Example:
```yaml
export AWS_PROFILE=dp-staging
```

#### Remote Allow security group rule already exists error

```sh
$ dp remote allow sandbox
[dp] allowing access to sandbox
Error: error adding rules to bastionSG: InvalidPermission.Duplicate: the specified rule "peer: X.X.X.X/32, TCP, from port: 22, to port: 22, ALLOW" already exists
        status code: 400, request id: 26a61345-8391-4c65-bfd7-4f0052892b6b
```

The error occurs when rules have previously been added and the command is run again.
Use `dp remote deny $env` to clear out existing rules and try again.

*This error should no longer appear* - the code should now avoid re-adding existing rules.
However, it is possible that the rule has been added with a description that does not match your username.
If so, you will have to use the AWS web UI/console to remove any offending security group rules.

### Advanced use

#### ssh commands

You can run ssh commands from the command-line, for example to determine the time on a given host:

```sh
$ dp ssh sandbox web 1 date
[...motd banner...]
[result of date command]
```

:warning: However, if you wish to include *flags* in the (remote) command, you must tell `dp` to stop looking for flags - use the `--` flag:

```sh
$ dp ssh develop web 1 -- ls -la
[...]
```

#### Manually configuring your IP or user

Optionally, (e.g. to avoid the program looking-up your IP),
you can use the `--ip` flag (or an environment variable `MY_IP`) to force the IP used when running `dp remote allow`.
For example:

```sh
dp remote --ip 192.168.11.22 allow sandbox
# or
MY_IP=192.168.11.22 dp remote allow sandbox
```

Similarly, use the `--user` flag to change the label attached to the IP that is put into (or removed from) the *allow* table.

```sh
dp remote --user MyColleaguesName --ip 192.168.44.55 --http-only allow develop
```

#### Remote allow extra ports

You can expand the allowed ports in your config for `publishing`, `web` or `bastion` with:

```yaml
environments:
  - name: example-environment
    extra-ports:
      publishing:
        - 80
```
#### AWS Command Line Access

Follow the guide in [dp](https://github.com/ONSdigital/dp/blob/main/guides/AWS_ACCOUNT_ACCESS.md)
## Releases

When creating new releases, please be sure to:

- update the version (tag)
- update the brew formula [in the tap](https://github.com/ONSdigital/homebrew-dp-cli).

## Sample config for `~/.aws/config`:

```
[profile dp-sandbox]
sso_start_url = https://ons.awsapps.com/start
sso_account_id = 1234556253 #replace this with correct account id
sso_role_name = AdministratorAccess
sso_region = eu-west-2
region = eu-west-2
```