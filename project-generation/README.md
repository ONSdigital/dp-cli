# Project Generation 

## What
This tool can be used to create projects of the following categories:
- generic-project
    - base-application 
        - api 
        - controller 
        - event-driven

## How to use
It is always beneficial to ensure you are using the most up to date version of the dp-utils tool. 
To update pull the latest changes and rebuild the tool like so:
```shell script
git pull; go build -o ./dp-utils; 
```

```shell script
./dp-utils generate-project
``` 

This tool can be used in conjunction with the repository creation tool, for further details read [COMPLETE_PROJECT_SETUP.md](COMPLETE_PROJECT_SETUP.md)

### Optional flags
Although these flags are optional, for most, if they are not provided then the user will be prompted for details.
- --name :              The name of the application, if Digital specific application it should be prepended with 'dp-'
- --go-version :        The version of Go the application should use (Not used on generic-projects)
- --project-location :  Location to generate project in
- --create-repository : Should a repository be created for the project, default no. Value can be y/Y/yes/YES/ or n/N/no/NO")
- --type :              Type of application to generate, values can be: 'generic-project', 'base-application', 'api', 'controller', 'event-driven'")

