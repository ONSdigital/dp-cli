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

The DP CLI uses Go Modules so requires a go version of **1.11** or later.

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

**Optional:**

 The following are only required for some functionality of this tool.

In order to use the `dp ssh` sub-command you will need:

- [`dp-setup`](https://github.com/ONSdigital/dp-setup) cloned locally:

  ```bash
  git clone git@github.com:ONSdigital/dp-setup
  ```

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

[ set paths for:

```yaml
    dp-setup-path:
    dp-hierarchy-builder-path:
    dp-code-list-scripts-path:

   set your `ssh-user:`
```

and if this is a first time setup, comment out production from environments, thus:

```yaml
    # - name: production
    #   profile: production
```

]

*Note*: **ssh-user** is actually your AWS account name. You should receive credentials as part of onboarding. If you do not have credentials yet, please ask a Tech Lead ([as documented](https://github.com/ONSdigital/dp/blob/main/guides/AWS_CREDENTIALS.md)).

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

`error creating group commands for env: develop: error fetching ec2: {"develop" "development"}: NoCredentialProviders: no valid providers in chain. Deprecated.`

[Ensure you have AWS credentials set up](https://github.com/ONSdigital/dp/blob/main/guides/AWS_CREDENTIALS.md).

If you do not want to set up separate profiles, another option is to not specify any profiles in your `~/.dp-cli-config.yml`. That way the default credentials will be used.

```yaml
environments:
  - name: production
    profile:
  - name: develop
    profile:
```

#### SSH/SCP command fails

```sh
$ dp ssh develop
ssh to develop
```

If the SSH or SCP command fails, ensure that the `dp remote allow` command has been run for the environment you want to connect to.

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

*This error should no longer appear* - the code should now avoid re-adding existing rules.
However, it is possible that the rule has been added with a description that does not match your username.
If so, you will have to use the AWS web UI/console to remove any offending security group rules.

### Advanced use

#### ssh commands

You can run ssh commands from the command-line, for example to determine the time on a given host:

```sh
$ dp ssh develop web 1 date
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
dp remote --ip 192.168.11.22 allow develop
# or
MY_IP=192.168.11.22 dp remote allow develop
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

## Releases

When creating new releases, please be sure to:

- update the version (tag)
- update the brew formula [in the tap](https://github.com/ONSdigital/homebrew-dp-cli).
