package appgen

// TODO add annotations on functions
// TODO process input from user on if lib or not
// TODO enable cloning and pushing to repo
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

type templateVars struct {
	Name      string
	Year      int
	GoVersion string
}

type application struct {
	templateVars templateVars
	pathToRepo   string
	name         string
	license      string
	port         string
	progType     ProgramType
}

type fileGen struct {
	path      string
	extension string
}

type ProgramType string

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
	GenericProgram  ProgramType = "generic-program"
	BaseApplication ProgramType = "base-application"
	API             ProgramType = "api"
	Controller      ProgramType = "controller"
	EventDriven     ProgramType = "event-driven"
)

func GenerateProject(appName, projectType, projectLocation, goVer string) error {
	ctx := context.Background()
	var err error
	appName, projectType, projectLocation, goVer, err = validateArguments(ctx, appName, projectType, projectLocation, goVer)
	if err != nil {
		log.Event(ctx, "error validating arguments for command", log.Error(err))
		return err
	}
	newApp := application{
		pathToRepo:   projectLocation,
		progType:     ProgramType(projectType),
		name:         appName,
		templateVars: populateTemplateModel(appName, goVer),
	}

	// If path has files in then purge them... but check with user first (prompt are you sure)
	isEmpty, err := IsEmptyDir(newApp.pathToRepo)
	if err != nil {
		return err
	}

	if !isEmpty {
		//prompt user
		maxUserInputAttempts := 3
		deleteContents := promptForConfirmation(ctx, "The directory "+newApp.pathToRepo+" was not empty would you "+
			"like to purge its contents, this will also remove any git files if present?", maxUserInputAttempts)

		if deleteContents {
			err := os.RemoveAll(newApp.pathToRepo)
			if err != nil {
				return err
			}
			err = os.MkdirAll(newApp.pathToRepo, os.ModePerm)
			if err != nil {
				return err
			}
		}
		fmt.Println("Path to generate files created")
	}

	initGoModules(ctx, newApp.pathToRepo, newApp.name)

	switch newApp.progType {
	case GenericProgram:
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
		log.Event(ctx, "unable to generate program due to unknown program type given", log.Error(err))
	}
	finaliseModules(ctx, newApp.pathToRepo)
	log.Event(ctx, "Project creation complete. Project can be found at "+newApp.pathToRepo)
	return nil
}

func validateArguments(ctx context.Context, unvalidatedName, unvalidatedType, unvalidatedProjectLocation, unvalidatedGoVersion string) (string, string, string, string, error) {
	var validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion string
	var err error
	if unvalidatedName == "unset" {
		validatedAppName, err = promptForInput(ctx, "Please specify the name of the application, if this is a "+
			"Digital publishing specific application it should be prepended with 'dp-'")
		if err != nil {
			return "", "", "", "", err
		}
	} else {
		validatedAppName = unvalidatedName
	}
	validatedProjectType, err = ValidateProjectType(ctx, unvalidatedType)

	if unvalidatedProjectLocation == "unset" {
		validatedProjectLocation, err = promptForInput(ctx, "Please specify a directory for the project to be created in")
		if err != nil {
			return "", "", "", "", err
		}
	} else {
		validatedProjectLocation = unvalidatedProjectLocation
	}
	if validatedProjectLocation[len(validatedProjectLocation)-1:] != "/" {
		validatedProjectLocation = validatedProjectLocation + "/"
	}

	if unvalidatedGoVersion == "unset" && validatedProjectType != "generic-program" {
		validatedGoVersion, err = promptForInput(ctx, "Please specify the version of GO to use")
		if err != nil {
			return "", "", "", "", err
		}
	} else {
		validatedGoVersion = unvalidatedGoVersion
	}

	return validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion, nil
}

func ValidateProjectType(ctx context.Context, unvalidatedType string) (validatedProjectType string, err error) {
	typeInputValid := false
	validTypes := []string{
		"generic-program", "base-application", "api", "controller", "event-driven",
	}
	for !typeInputValid {
		if !stringInSlice(unvalidatedType, validTypes) {
			typeInputValid = false
			unvalidatedType, err = promptForInput(ctx, "Please specify the project type. This can have one of the "+
				"following values: 'generic-program', 'base-application', 'api', 'controller', 'event-driven'")
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

func populateTemplateModel(name, goVer string) templateVars {
	// UTC to avoid any sketchy BST timing
	year := time.Now().UTC().Year()
	return templateVars{
		Name:      name,
		Year:      year,
		GoVersion: goVer,
	}
}

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

func initGoModules(ctx context.Context, pathToRepo, name string) {
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error  initialising go modules", log.Error(err))
	}
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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

func finaliseModules(ctx context.Context, pathToRepo string) () {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during go build step", log.Error(err))
	}
}

func (a application) createGenericContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+".github", os.ModePerm)
}

func (a application) createApplicationContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"ci/scripts", os.ModePerm)
}

func (a application) createAPIContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"api", os.ModePerm)
}

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

func (a application) createEventDrivenContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+"event", os.ModePerm)
}

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

func (a application) generateBatchOfFileTemplates(filesToGen []fileGen) error {
	for _, fileToGen := range filesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

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

	err = tmpl.Execute(writer, a.templateVars)
	if err != nil {
		return err
	}
	return nil
}
