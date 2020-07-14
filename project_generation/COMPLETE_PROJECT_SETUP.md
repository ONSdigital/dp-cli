# Repo creation with project templating

The project templating tool can be used to also create a repository.
____

## Prerequisites

### Be up to date

It is always beneficial to ensure you are using the most up to date version of the dp-cli tool.
To update pull the latest changes and rebuild the tool like so:

```shell script
git pull && make install
```

### Have required access

Ensure the correct access permissions have been granted and a personal access token has been generated.
Further details can be found at [README.md](../repository_creation/README.md)
____
There are three ways of creating a repository and using the boilerplate generation.

### First method, use prompts

`generate-project` with the option `--create-repository` set to `yes`
(note that `y` is also accepted - this is not case sensitive).

```shell script
dp generate-project --create-repository yes
```

If this method is chosen then there wil be numerous prompts such as the name of the application,
location to build it and the type of project it should boilerplate.
____

### Second method - use command-line arguments

The second way to use this tool is to provide information via the command-line as options like so

```shell script
dp generate-project --create-repository yes --name {name-of-repository} --project-location {location} --type {project-type}
```

If this method is chosen then there will be less prompts during the creation of the project
____

### Third method - use repo-creation tool then boilerplate generation tool

The final way to complete this is to use the repo creation tool independently of the templating tool.
This is useful if for some reason the name of the repository is different to the name of the
project - however this should be avoided.

Note: _`generate-project` can also be called after `create-repo` the order does not matter_

```shell script
dp generate-project
```

After running follow the prompts then run the repo-creation tool

```shell script
dp create-repo github
```

And follow the prompts, alternatively provide command-line arguments.
