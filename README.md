# dp-cli

> [!WARNING]
> This tool is primarily for internal use at ONS but feel free to fork for your own use.
>
> If you notice any bugs/issues please open a GitHub issue.

Command-line client providing *handy helper tools* for the ONS Dissemination Platform software engineering team

## Getting started

If using macOS, you can install using `brew`:

- Create tap

  ```shell
  brew tap ONSdigital/homebrew-dp-cli git@github.com:ONSdigital/homebrew-dp-cli
  ```

- Run brew install

   ```shell
   brew install dp-cli
   ```

### Prerequisites

The cli tool will do its best to check you have the required supporting tools installed, but you will need to have the following installed to use the tool:

- **aws cli** - Either `brew install awscli` or follow the [AWS docs](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2-mac.html)
- **aws session manager plugin** - Either `brew install --cask session-manager-plugin` or follow the [AWS docs](https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html#install-plugin-macos)
- **socat** - Either `brew install socat`. Only required for `dp eks session --legacy-sudo` (the legacy sudo tunnel path). It is **not** required in the default no-sudo mode (see [EKS session tunnels](#eks-session-tunnels)).

#### Optional but common requirements

The following are only required for some common functionality of this tool.

In order to use the `dp ssh` sub-command you will need:

- [`dp-setup`](https://github.com/ONSdigital/dp-setup) cloned locally:

  ```shell
  git clone git@github.com:ONSdigital/dp-setup
  ```

- [`dp-ci`](https://github.com/ONSdigital/dp-ci) cloned locally:

  ```shell
  git clone git@github.com:ONSdigital/dp-ci
  ```

- [`dp-nisra-infrastructure`](https://github.com/ONSdigital/dp-nisra-infrastructure) cloned locally:

  ```shell
  git clone git@github.com:ONSdigital/dp-nisra-infrastructure
  ```

Note: Make sure your repo's are on the right branches and are up-to-date:

- `dp-setup` is on the `awsb` (or `main`) branch
- `dp-ci` is on the `main` branch
- `dp-nisra-infrastructure` is on the `develop` branch

This is necessary because they have the required SSH configuration and the relevant ansible inventories.

#### Optional and less common CMD requirements

In order to use the `dp import cmd` sub-command (e.g. when you are using **Neo4j**; however, `import` is currently *not needed* if you are using Neptune) you will need:

- [`dp-code-list-scripts`](https://github.com/ONSdigital/dp-code-list-scripts) cloned locally:

  ```shell
  git clone git@github.com:ONSdigital/dp-code-list-scripts
  ```

- [`dp-hierarchy-builder`](https://github.com/ONSdigital/dp-hierarchy-builder) cloned locally:

  ```shell
  git clone git@github.com:ONSdigital/dp-hierarchy-builder
  ```

### Tools

To run some of our tests you will need additional tooling:

#### Audit

We use `dis-vulncheck` to do auditing, which you will [need to install](https://github.com/ONSdigital/dis-vulncheck).

### Configuration

Configuration is defined in a YAML file:

- By default the CLI expects the config file to be `~/.dp-cli-config.yml`.
- The config file location can be customised by setting the `DP_CLI_CONFIG` environment variable to your chosen path.

The [sample config file](./config/example_config.yml) should be copied and tailored to suit you. For example:

```shell
cp -i config/example_config.yml ~/.dp-cli-config.yml
vi ~/.dp-cli-config.yml
```

update the paths and `user-name`:

```yaml
    dp-setup-path: path to your local dp-setup
    dp-ci-path: path to your local dp-ci
    dp-nisra-path: path to dp-nisra-infrastructure
    dp-hierarchy-builder-path: path to your local dp-hierarchy-builder-path
    dp-code-list-scripts-path: path to your local dp-code-list-scripts-path
    dp-cli-path: path to your local dp-cli
    user-name: Your first and last name concatenated eg. JaneBloggs"
```

You can uncomment more `environments` values as and when you get access to them.

### AWS Profile Setup

The CLI uses role-based AWS profiles to enforce least-privilege access. Each environment needs up to three profiles in your `~/.aws/config`:

```ini
# Sandbox - view only (default for read operations)
[profile dp-sandbox-view-only]
sso_start_url = https://<your-org>.awsapps.com/start
sso_region = eu-west-2
sso_account_id = <account-id>
sso_role_name = dis_view_only_access
region = eu-west-2
output = json

# Sandbox - engineer (for write operations, ssh, terraform apply)
[profile dp-sandbox-engineer]
sso_start_url = https://<your-org>.awsapps.com/start
sso_region = eu-west-2
sso_account_id = <account-id>
sso_role_name = dis_engineer_access
region = eu-west-2
output = json

# Sandbox - admin (break-glass only)
[profile dp-sandbox-admin]
sso_start_url = https://<your-org>.awsapps.com/start
sso_region = eu-west-2
sso_account_id = <account-id>
sso_role_name = dis_admin_access
region = eu-west-2
output = json
```

Repeat for each environment (`dp-bleed-dev`, `dp-staging`, `dp-prod`, `dp-ci`, `dp-nisra-dev`, `dp-nisra-prod`).

The `profile-suffixes` and `command-privileges` sections in `~/.dp-cli-config.yml` control which profile the CLI uses for each command:

- **view** (`-view-only`) — read-only operations: `remote allow/deny`, `eks session`, instance listing
- **engineer** (`-engineer`) — write operations: `ssh`, `scp`, terraform apply
- **admin** (`-admin`) — break-glass: full admin access

Commands that require elevated access will validate credentials before executing and show guidance if access is denied.

For staging, production, and CI accounts, engineer and admin access requires approval via TAPS (your organisation's temporary access provisioning service).

## Binary build and run

```shell
git clone git@github.com:ONSdigital/dp-cli.git
```

```shell
make install
dp
```

> [!IMPORTANT]
> `dp-cli` uses Go Modules and **must** be cloned to a location outside of your `$GOPATH`.

- If you get:

  `command not found: dp`

  Then either edit your `~/.zshrc` file to have the correct path *or* do:

  ```shell
  echo 'export PATH="$GOPATH/bin:$PATH"' >> ~/.zshrc
  ```

  and restart the terminal

Or to build a binary in this directory:

```shell
make build
./dp
```

You should be presented with a help menu similar to:

```text
dp is a command-line client providing handy helper tools for ONS Dissemination Platform software engineers

Usage:
  dp [command]

Available Commands:
  clean            Delete data from your local environment
  completion       Generate the autocompletion script for the specified shell
  create-repo      Creates a new repository with the typical Dissemination Platform configurations 
  eks              EKS cluster management commands
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

## EKS session tunnels

`dp eks session start <ENV>` opens secure, auditable tunnels to the EKS API
servers via the SSM tunnel box, so `kubectl`, `k9s` and Terraform can reach the
clusters. The tunnel traffic is carried over an SSM port-forward to a local high
port (9443-9500).

There are two tunnel modes.

### No-sudo mode (default)

No-sudo mode is the **default** and requires **no `sudo` and no `socat`**. Instead
of re-presenting the tunnel on port 443 with the real hostname, dp-cli points the
kubeconfig `cluster` block directly at the local port and uses the
`tls-server-name` field to keep TLS validation working:

```yaml
cluster:
  server: https://127.0.0.1:<localPort>
  certificate-authority-data: <cluster CA>   # unchanged
  tls-server-name: <real EKS endpoint hostname>
```

`tls-server-name` tells `kubectl` to send the real EKS hostname as the TLS SNI and
validate the server certificate against it, while actually connecting to
`127.0.0.1:<localPort>`. TLS is still **fully validated** — there is no
`insecure-skip-tls-verify`. Because there is no `socat`, no `/etc/hosts` edit and
no loopback alias, no elevated privileges are needed.

This is what you get by default:

```shell
dp eks session start sandbox
```

Notes (especially useful on managed devices without permanent admin rights):

- You do **not** need `sudo` — you are never prompted for a password.
- You do **not** need to install `socat` (this was the last remaining tool that
  could not be cleanly installed without conda).
- You do **not** need a JIT admin request to open a tunnel.

### Legacy (sudo) mode

The previous behaviour is still available behind the `--legacy-sudo` flag. It uses
`socat` to bind the loopback address on port 443, adds an `/etc/hosts` entry
mapping the EKS endpoint to that loopback address, and creates a loopback alias.
These operations require `sudo`, and `socat` must be installed.

```shell
dp eks session start sandbox --legacy-sudo
```

Both modes are interoperable — `dp eks session status` and `dp eks session stop`
detect the mode each active tunnel was created in and handle it accordingly. When
you start a no-sudo tunnel, dp-cli will not touch `sudo`, `socat`, or your
`/etc/hosts` at all; if it finds a leftover *legacy* tunnel from a previous run it
will leave the privileged parts in place (warning you to run
`dp eks session stop`) rather than prompting for a password.

### Configuration

The `eks:` section of `~/.dp-cli-config.yml` accepts optional overrides (defaults shown):

```yaml
eks:
  tunnel-box-role-tag: "session-tunnel-box"  # tag used to discover the SSM tunnel box
  cluster-access-tag: "ssm-tunnel-access"    # cluster tag opting a cluster into tunnel access
  state-dir: "/tmp/eks-tunnels-ssm"          # where local tunnel state is stored
  base-port: 9443                            # local SSM port-forward range (inclusive)
  max-port: 9500
```

- **state-dir**: local tunnel state is stored here as one `<cluster>.json` file per
  active tunnel. The default is under `/tmp`, so it is ephemeral and cleared on
  reboot (which suits transient tunnel state). Set a persistent path (e.g.
  `~/.dp-cli/eks-tunnels`) if you want state to survive reboots. `~` is expanded.
- **base-port / max-port**: one port from this inclusive range is used per active
  cluster tunnel. If a port is already in use it is skipped automatically. If the
  whole range is exhausted, that cluster's tunnel is skipped with a warning (other
  clusters still proceed). Widen the range if you routinely open many tunnels, or
  move it if `9443` clashes with another local service.


## Common issues

### Credentials error

1. If sandbox/prod/staging are not in the dp cli output try unsetting `AWS_REGION` and `AWS_DEFAULT_REGION`

2. `SSOProviderInvalidToken: the SSO session has expired or is invalid`

    If you see the above error, you need to re-authenticate with sign-in information

    Try: `dp remote login`

3. `error fetching ec2: {Name:sandbox Profile:dp-sandbox SSHUser:ubuntu Tag: CI:false ExtraPorts:{Bastion:[] Publishing:[] Web:[]}}: MissingRegion: could not find region configuration`

    check that you have the correct AWS profile names in your `~/.aws/config` file (`dp-sandbox`, `dp-staging`, `dp-prod`, `dp-ci`).
    A sample config for `~/.aws/config` is included at the end of this guide as a reference.

4. `Error: no security groups matching environment: "sandbox" with name "sandbox - bastion"`

    check `~/.aws/credentials` and remove any profile information added for `dp-sandbox`, `dp-staging` and `dp-prod` as this is not needed

    If you do not want to set up separate profiles, another option is to not specify any profiles in your `~/.dp-cli-config.yml`. That way the default credentials will be used.

    ```yaml
    environments:
      - name: prod
        profile:
      - name: staging
        profile:
    ```

### SSH/SCP command fails

```shell
$ dp ssh sandbox
ssh to sandbox
...
```

If the SSH or SCP command fails, ensure that the `dp remote allow` command has been run for the environment you want to connect to.

### Remote Allow security group error

`Error: no security groups matching environment: "sandbox" with name "sandbox - bastion"`

Ensure you have `region=eu-west-2` in your AWS configuration.

Depending on the command you're trying to run, and what you're trying to access:

- ensure your `AWS_PROFILE` is set correctly
- and there is no dp-prod/dp-sandbox/dp-ci config in the `~/.aws/credentials` file.

Example:

```yaml
export AWS_PROFILE=dp-staging
```

### Remote Allow security group rule already exists error

```shell
$ dp remote allow sandbox
[dp] allowing access to sandbox
Error: error adding rules to bastionSG: InvalidPermission.Duplicate: the specified rule "peer: X.X.X.X/32, TCP, from port: 22, to port: 22, ALLOW" already exists
        status code: 400, request id: 26a61345-8391-4c65-bfd7-4f0052892b6b
```

The error occurs when rules have previously been added and the command is run again.
Use (e.g.) `dp remote deny sandbox` to clear out existing rules and try again.

> [!NOTE]
> *This error should no longer appear* - the code should now avoid re-adding existing rules.
> However, it is possible that the rule has been added with a description that does not match your username.
> If so, you will have to use the AWS web UI/console to remove any offending Security Group rules.

## Advanced use

### ssh commands

You can run ssh commands from the command-line, for example to determine the time on a given host:

```shell
$ dp ssh sandbox web 1 date
[...motd banner...]
[result of date command when run on remote host]
```

:warning: However, if you wish to include *flags* in the (remote) command, you must tell `dp` to stop looking for flags - use the `--` flag:

```shell
$ dp ssh sandbox web 1 -- ls -la
[...]
$ dp ssh sandbox web_mount 1 --to 2 -- ls -la
# runs `ls -la` on web_mount 1 and 2
$ dp ssh sandbox web 1 --to 0 -- ls -la
# runs `ls -la` on ALL web boxes
```

### Manually configuring your IP or user

Optionally, (e.g. to avoid the program looking-up your IP),
you can use the `--ip` flag (or an environment variable `MY_IP`) to force the IP used when running `dp remote allow`.
For example:

```sh
dp remote --ip 192.168.11.22 allow sandbox
# or
MY_IP=192.168.11.22 dp remote allow sandbox
```

Similarly, use the `--user` flag to change the label attached to the IP that is put into (or removed from) the *allow* table.

```shell
dp remote --user MyColleaguesName --ip 192.168.44.55 --http-only allow sandbox
```

### Remote allow extra ports

You can expand the allowed ports in your config for `publishing`, `web` or `bastion` with:

```yaml
environments:
  - name: example-environment
    extra-ports:
      publishing:
        - 80
```

### AWS Command Line Access

Follow the guide in [dp](https://github.com/ONSdigital/dp/blob/main/guides/AWS_ACCOUNT_ACCESS.md)

## Releases

When creating new releases, please be sure to:

- update the version (tag)
- update the brew formula [in the tap](https://github.com/ONSdigital/homebrew-dp-cli).

## Sample config for `~/.aws/config`

```ini
[profile dp-sandbox]
sso_start_url  = https://ons.awsapps.com/start
sso_account_id = 1234556253   # replace this with correct account id (from above URL)
sso_role_name  = AdministratorAccess
sso_region     = eu-west-2
region         = eu-west-2
```

Repeat the above for any other environments (with the appropriate profile-name and account-id changes).

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

## License

Copyright © 2024, Office for National Statistics <https://www.ons.gov.uk>

Released under MIT license, see [LICENSE](LICENSE.md) for details.
