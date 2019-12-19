package projectgeneration

import (
	"bufio"
	"context"
	"fmt"
	"github.com/ONSdigital/log.go/log"
	"os"
	"text/template"
)

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
	{
		path:      "Dockerfile.concourse",
		extension: "",
	},

	// TODO Make file
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

func ValidateMandatoryArguments(nameOfApp, projType, projectLocation, goVer string) error {
	ctx := context.Background()
	var err error
	projType, err = ValidateProjectType(ctx, projType)
	if err != nil {
		log.Event(ctx, "error unable to validate project type", log.Error(err))
		return err
	}
	nameOfApp, err = ValidateAppName(ctx, nameOfApp)
	if err != nil {
		log.Event(ctx, "error unable to validate name of application", log.Error(err))
		return err
	}
	projectLocation, err = ValidateProjectLocation(ctx, projectLocation, nameOfApp, true)
	if err != nil {
		log.Event(ctx, "error unable to validate project location", log.Error(err))
		return err
	}
	return nil
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
	tmpl := template.Must(template.ParseFiles("./project-generation/content/templates/" + fileToGen.path + ".tmpl"))

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
