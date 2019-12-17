package projectgeneration

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ONSdigital/log.go/log"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

// ~TODO add annotations on functions~
// ~TODO create README.md~
// ~TODO enable cloning and pushing to repo~
// ~TODO rename file~
// TODO process input from user on if lib or not
// TODO validateArguments better

type templateModel struct {
	Name      string
	Year      int
	GoVersion string
}

type application struct {
	templateModel templateModel
	pathToRepo    string
	name          string
	license       string
	port          string
	projectType   ProjectType
}

type fileGen struct {
	path      string
	extension string
}

type ProjectType string

var genericFiles = []fileGen{
	{
		path:      "readme",
		extension: ".md",
	},
	{
		path:      "contributing",
		extension: ".md",
	},
	{
		path:      "license",
		extension: ".md",
	},
	{
		path:      ".gitignore",
		extension: "",
	},
	{
		path:      ".github/PULL_REQUEST_TEMPLATE",
		extension: ".md",
	},
	{
		path:      ".github/ISSUES_TEMPLATE",
		extension: ".md",
	},
}

var applicationFiles = []fileGen{
	{
		path:      "ci/build",
		extension: ".yml",
	},
	{
		path:      "ci/unit",
		extension: ".yml",
	},
	{
		path:      "ci/scripts/build",
		extension: ".sh",
	},
	{
		path:      "ci/scripts/unit",
		extension: ".sh",
	},

	// TODO Make file
	// TODO concourse.docker
	// TODO {appname}.Nomad
	// TODO Config
	// TODO Main
}

var controllerFiles = []fileGen{
	{
		path:      "handlers/handlers",
		extension: ".go",
	},
	{
		path:      "handlers/handlers_test",
		extension: ".go",
	},
	{
		path:      "routes/routes",
		extension: ".go",
	},
	{
		path:      "routes/routes_test",
		extension: ".go",
	},
	{
		path:      "mapper/mapper",
		extension: ".go",
	},
	{
		path:      "mapper/mapper_test",
		extension: ".go",
	},
}

var apiFiles = []fileGen{
	{
		// TODO Swagger spec
		// TODO api/Api.go
		// TODO api/Api_test.go
		// TODO api/Hello.go
		// TODO api/hello_test.go
	},
}

var eventFiles = []fileGen{
	{
		// Todo event/
		// TODO Event
		// TODO Consumer
		// TODO Consumer_test
		// TODO handler
	},
}

const (
	GenericProject  ProjectType = "generic-project"
	BaseApplication ProjectType = "base-application"
	API             ProjectType = "api"
	Controller      ProjectType = "controller"
	EventDriven     ProjectType = "event-driven"
)

// GenerateProject is the entry point into generating a project
func GenerateProject(appName, projType, projectLocation, goVer string, repositoryCreated bool) error {
	ctx := context.Background()
	var err error
	appName, projType, projectLocation, goVer, err = validateArguments(ctx, repositoryCreated, appName, projType, projectLocation, goVer)
	if err != nil {
		log.Event(ctx, "error validating arguments for command", log.Error(err))
		return err
	}
	newApp := application{
		pathToRepo:    projectLocation + appName + "/",
		projectType:   ProjectType(projType),
		name:          appName,
		templateModel: populateTemplateModel(appName, goVer),
	}

	initGoModules(ctx, newApp.pathToRepo, newApp.name)

	switch newApp.projectType {
	case GenericProject:
		err := newApp.generateGenericContent()
		if err != nil {
			return err
		}
	case BaseApplication:
		err := newApp.generateApplicationContent()
		if err != nil {
			return err
		}
	case API:
		err := newApp.generateApiContent()
		if err != nil {
			return err
		}
	case Controller:
		err := newApp.generateControllerContent()
		if err != nil {
			return err
		}
	case EventDriven:
		err := newApp.generateEventDrivenContent()
		if err != nil {
			return err
		}
	default:
		log.Event(ctx, "unable to generate project due to unknown project type given", log.Error(err))
	}
	finaliseModules(ctx, newApp.pathToRepo)
	log.Event(ctx, "Project creation complete. Project can be found at "+newApp.pathToRepo)
	return nil
}

// validateArguments will ensure that all user input via flags have been provided and if not prompt for them to be and
// keep prompting until all input meets validation criteria
func validateArguments(ctx context.Context, repositoryCreated bool, unvalidatedName, unvalidatedType, unvalidatedProjectLocation, unvalidatedGoVersion string) (string, string, string, string, error) {
	var validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion string
	var err error
	validatedAppName, err = ValidateAppName(ctx, unvalidatedName)
	validatedProjectType, err = ValidateProjectType(ctx, unvalidatedType)
	if err != nil {
		return "", "", "", "", err
	}
	offerPurge := !repositoryCreated
	validatedProjectLocation, err = ValidateProjectLocation(ctx, unvalidatedProjectLocation, validatedAppName, offerPurge)
	if err != nil {
		return "", "", "", "", err
	}

	if unvalidatedGoVersion == "unset" && validatedProjectType != "generic-project" {
		validatedGoVersion, err = promptForInput(ctx, "Please specify the version of GO to use")
		if err != nil {
			return "", "", "", "", err
		}
	} else {
		validatedGoVersion = unvalidatedGoVersion
	}

	return validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion, nil
}

// ValidateAppName will ensure that the app name has been provided and is acceptable, if not it will keep
// prompting until it is
func ValidateAppName(ctx context.Context, unvalidatedAppName string) (validatedAppName string, err error) {
	if unvalidatedAppName == "unset" {
		validatedAppName, err = promptForInput(ctx, "Please specify the name of the application, if this is a "+
			"Digital publishing specific application it should be prepended with 'dp-'")
		if err != nil {
			return validatedAppName, err
		}
	} else {
		validatedAppName = unvalidatedAppName
	}
	return validatedAppName, err
}

// ValidateProjectType will ensure that the project type provided by the users is one that can be boilerplate
func ValidateProjectType(ctx context.Context, unvalidatedType string) (validatedProjectType string, err error) {
	typeInputValid := false
	validTypes := []string{
		"generic-project", "base-application", "api", "controller", "event-driven",
	}
	for !typeInputValid {
		if !stringInSlice(unvalidatedType, validTypes) {
			typeInputValid = false
			unvalidatedType, err = promptForInput(ctx, "Please specify the project type. This can have one of the "+
				"following values: 'generic-project', 'base-application', 'api', 'controller', 'event-driven'")
			if err != nil {
				return validatedProjectType, err
			}
		} else {
			typeInputValid = true
			validatedProjectType = unvalidatedType
		}
	}
	return validatedProjectType, err
}

// ValidateProjectLocation will ensure that the projects location has been provided and is acceptable.
// It will ensure the directory exists and has the option to offer a purge of files at that location
func ValidateProjectLocation(ctx context.Context, unvalidatedProjectLocation, appName string, offerPurge bool) (validatedProjectLocation string, err error) {
	if unvalidatedProjectLocation == "unset" {
		validatedProjectLocation, err = promptForInput(ctx, "Please specify a directory for the project to be created in")
		if err != nil {
			return validatedProjectLocation, err
		}
	} else {
		validatedProjectLocation = unvalidatedProjectLocation
	}
	if validatedProjectLocation[len(validatedProjectLocation)-1:] != "/" {
		validatedProjectLocation = validatedProjectLocation + "/"
	}
	if offerPurge {
		err = offerPurgeProjDestination(ctx, validatedProjectLocation, appName)
		if err != nil {
			return validatedProjectLocation, err
		}
	}
	return validatedProjectLocation, err
}

// offerPurgeProjDestination will offer the user an option to purge the contents at a given location
func offerPurgeProjDestination(ctx context.Context, projectLoc, appName string) error {
	fmt.Println("offerPurgeProjDestination was hit")
	if _, err := os.Stat(projectLoc + appName); os.IsNotExist(err) {
		err = os.MkdirAll(projectLoc+appName, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// If path has files in then purge them... but check with user first (prompt are you sure)
	isEmpty, err := IsEmptyDir(projectLoc + appName)
	if err != nil {
		return err
	}

	if !isEmpty {
		//prompt user
		maxUserInputAttempts := 3
		deleteContents := promptForConfirmation(ctx, "The directory "+projectLoc+appName+" was not empty would you "+
			"like to purge its contents, this will also remove any git files if present?", maxUserInputAttempts)

		if deleteContents {
			err := os.RemoveAll(projectLoc + appName)
			if err != nil {
				return err
			}
			err = os.MkdirAll(projectLoc+appName, os.ModePerm)
			if err != nil {
				return err
			}
		}
		fmt.Println("Path to generate files created")
	}
	return nil
}

// IsEmptyDir will check if a given directory is empty or not
func IsEmptyDir(name string) (isEmptyDir bool, err error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}

	defer func() {
		// Note 'cerr' used to check that no other error happened prior to closing and that original error is not disguised
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

// populateTemplateModel will populate the templating model with variables that can be used in templates
func populateTemplateModel(name, goVer string) templateModel {
	// UTC to avoid any sketchy BST timing
	year := time.Now().UTC().Year()
	return templateModel{
		Name:      name,
		Year:      year,
		GoVersion: goVer,
	}
}

// promptForConfirmation will prompt for yes/no style answers on command line
func promptForConfirmation(ctx context.Context, prompt string, maxInputAttempts int) bool {
	reader := bufio.NewReader(os.Stdin)

	for ; maxInputAttempts > 0; maxInputAttempts-- {
		fmt.Printf("%s [y/n]: ", prompt)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Event(ctx, "error reading user input ", log.Error(err))
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}

	return false
}

func promptForInput(ctx context.Context, prompt string) (string, error) {
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(prompt)
	scanner.Scan()
	input = scanner.Text()
	if scanner.Err() != nil {
		log.Event(ctx, "project creation failed", log.Error(scanner.Err()))
		return "", scanner.Err()
	}
	return input, nil
}

// initGoModules will initialise the go modules for a project at a given directory
func initGoModules(ctx context.Context, pathToRepo, name string) {
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error initialising go modules", log.Error(err))
	}
}

// stringInSlice will check if a string as a complete word appears within a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

// finaliseModules will run go build ./... to generate go modules dependency management files
func finaliseModules(ctx context.Context, pathToRepo string) () {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during go build step", log.Error(err))
	}
}

// createGenericContentDirectoryStructure will create child directories for Generic content at a given path
func (a application) createGenericContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+".github", os.ModePerm)
}

// createApplicationContentDirectoryStructure will create child directories for Application content at a given path
func (a application) createApplicationContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"ci/scripts", os.ModePerm)
}

// createAPIContentDirectoryStructure will create child directories for API content at a given path
func (a application) createAPIContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"api", os.ModePerm)
}

// createControllerContentDirectoryStructure will create child directories for Controller content at a given path
func (a application) createControllerContentDirectoryStructure() error {
	err := os.MkdirAll(a.pathToRepo+"handlers", os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(a.pathToRepo+"routes", os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(a.pathToRepo+"mappers", os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// createEventDrivenContentDirectoryStructure will create child directories for Event Driven content at a given path
func (a application) createEventDrivenContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"event", os.ModePerm)
}

// generateGenericContent will create all files for Generic content
func (a application) generateGenericContent() error {
	fmt.Println("function generateGenericContent hit")
	err := a.createGenericContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(genericFiles)
	if err != nil {
		return err
	}

	return nil
}

// generateApplicationContent will create all files for Application content
func (a application) generateApplicationContent() error {
	err := a.generateGenericContent()
	if err != nil {
		return err
	}

	err = a.createApplicationContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(applicationFiles)
	if err != nil {
		return err
	}

	return nil
}

// generateApiContent will create all files for API content
func (a application) generateApiContent() error {
	err := a.generateApplicationContent()
	if err != nil {
		return err
	}

	err = a.createAPIContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(apiFiles)
	if err != nil {
		return err
	}

	return nil
}

// generateControllerContent will create all files for Controller content
func (a application) generateControllerContent() error {
	err := a.generateApplicationContent()
	if err != nil {
		return err
	}

	err = a.createControllerContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(controllerFiles)
	if err != nil {
		return err
	}

	return nil
}

// generateEventDrivenContent will create all files for Event-Driven content
func (a application) generateEventDrivenContent() error {
	err := a.generateApplicationContent()
	if err != nil {
		return err
	}

	err = a.createEventDrivenContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(eventFiles)
	if err != nil {
		return err
	}

	return nil
}

// generateBatchOfFileTemplates will generate a batch of files from templates
func (a application) generateBatchOfFileTemplates(filesToGen []fileGen) error {
	for _, fileToGen := range filesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

// generateFileFromTemplate will generate a single file from templates
func (a application) generateFileFromTemplate(fileToGen fileGen) (err error) {
	f, err := os.Create(a.pathToRepo + fileToGen.path + fileToGen.extension)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	tmpl := template.Must(template.ParseFiles("./app-generation/content/templates/" + fileToGen.path + ".tmpl"))

	defer func() {
		ferr := writer.Flush()
		if err == nil {
			err = ferr
		}
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	err = tmpl.Execute(writer, a.templateModel)
	if err != nil {
		return err
	}
	return nil
}
