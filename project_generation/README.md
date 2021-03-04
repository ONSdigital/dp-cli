# Project Generation 

## What
This tool can be used to create projects of the following types:
  - [generic-project](#generic-project)
  - [base-application](#base-application)
  - [api](#api)
  - [controller](#controller)
  - [event-driven](#event-driven])
  - [library](#library)

## How to use
It is always beneficial to ensure you are using the most up to date version of the dp-cli tool. 
To update pull the latest changes and rebuild the tool like so:
```shell script
git pull; make install; 
```

```shell script
dp generate-project
``` 

This tool can be used in conjunction with the repository creation tool, for further details read [COMPLETE_PROJECT_SETUP.md](COMPLETE_PROJECT_SETUP.md)

### Optional flags
Although these flags are optional, for most, if they are not provided then the user will be prompted for details.
- --name :              The name of the application, if Digital specific application it should be prepended with 'dp-'
- --go-version :        The version of Go the application should use (Not used on generic-projects)
- --project-location :  Location to generate project in
- --create-repository : Should a repository be created on GitHub for the project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
- --type :              Type of application to generate, values can be: 'generic-project', 'base-application', 'api', 'controller', 'event-driven', 'library'")

### Example output
The project generation command has been used to create example outputs of the various types of project. These can be found
in the [dp-hello-world repository](https://github.com/ONSdigital/dp-hello-world). This provides a place where issues and
discussions around the content of the base projects can be discussed and agreed upon. Once agreed upon there, the
changes need to be applied to the templates in this repository and the example output can be regenerated with the new
version of this tool, ready for further improvements.

### What is created?

#### Generic Project
This option creates:
  - CONTRIBUTING.md
  - LICENSE.md
  - README.md

#### Base Application
This option creates:
  - everything in generic-project plus
  - ci folder
  - config folder
  - Dockerfile.concourse
  - go.mod
  - Makefile
  - go.sum
  - <repo_name>.nomad

#### API
This option creates:
  - everything in base-application plus
  - main.go
  - service folder
  - api folder
  - swagger.yaml

#### Controller
This option creates:
  - everything in base-application plus
  - handlers folder
  - mapper folder
  - routes

#### Event-Driven
This option creates:
  - everything in base-application plus
  - event folder 
  - schema
  - service
  - cmd

#### Library
This option creates:
  - everything in generic-project plus
  - ci folder
  - makefile
  - go.mod