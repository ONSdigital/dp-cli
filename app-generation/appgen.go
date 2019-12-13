package appgen

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/ONSdigital/go-ns/log"
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

type fileGenerationVars struct {
	filePath      string
	fileExtension string
}

type ProgramType string

const (
	GenericProgram  ProgramType = "generic-program"
	BaseApplication ProgramType = "base-application"
	API             ProgramType = "api"
	Controller      ProgramType = "controller"
	EventDriven     ProgramType = "event-driven"
)

func GenerateApp(appName, projectType, projectLocation, goVer string) error {
	appName, projectType, projectLocation, goVer = validateArguments(appName, projectType, projectLocation, goVer)
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
		deleteContents := promptForConfirmation("The directory "+newApp.pathToRepo+" was not empty would you "+
			"like to purge its contents, this will also remove any git files if present?", maxUserInputAttempts)

		if deleteContents {
			os.RemoveAll(newApp.pathToRepo)
			os.MkdirAll(newApp.pathToRepo, os.ModePerm)
		}
	}

	initGoModules(newApp.pathToRepo, newApp.name)

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
		log.Error(errors.New("unable to generate program due to unknown program type given"), nil)
	}
	//finaliseModules(pathToRepo)
	log.Info("Jobs d'ne. Project can be found at "+newApp.pathToRepo, nil)
	return nil
}

func validateArguments(unvalidatedName, unvalidatedType, unvalidatedProjectLocation, unvalidatedGoVersion string) (string, string, string, string) {
	var validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion string
	validTypes := []string{
		"generic-program", "base-application", "api", "controller", "event-driven",
	}
	if unvalidatedName == "unset" {
		validatedAppName = promptForInput("Please specify the name of the application, if this is a " +
			"Digital publishing specific application it should be prepended with 'dp-'")
	} else {
		validatedAppName = unvalidatedName
	}
	typeInputValid := false
	for !typeInputValid {
		if !stringInSlice(unvalidatedType, validTypes) {
			typeInputValid = false
			unvalidatedType = promptForInput("Please specify the project type. This can have one of the " +
				"following values: 'generic-program', 'base-application', 'api', 'controller', 'event-driven'")
		} else {
			typeInputValid = true
			validatedProjectType = unvalidatedType
		}
	}

	if unvalidatedProjectLocation == "unset" {
		validatedProjectLocation = promptForInput("Please specify a directory for the project to be created in")
	} else {
		validatedProjectLocation = unvalidatedProjectLocation
	}
	if validatedProjectLocation[len(validatedProjectLocation)-1:] != "/" {
		validatedProjectLocation = validatedProjectLocation + "/"
	}

	if unvalidatedGoVersion == "unset" {
		validatedGoVersion = promptForInput("Please specify the version of GO to use")
	} else {
		validatedGoVersion = unvalidatedGoVersion
	}

	return validatedAppName, validatedProjectType, validatedProjectLocation, validatedGoVersion
}
func IsEmptyDir(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

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

func promptForConfirmation(prompt string, maxInputAttempts int) bool {
	reader := bufio.NewReader(os.Stdin)

	for ; maxInputAttempts > 0; maxInputAttempts-- {
		fmt.Printf("%s [y/n]: ", prompt)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Error(err, nil)
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

func initGoModules(pathToRepo, name string) {
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Error(err, nil)
	}
}

func promptForInput(prompt string) string {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSuffix(input, "\n")
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (a application) generateGenericContent() error {
	err := a.generateReadMe()
	if err != nil {
		return err
	}
	err = a.generateContributionGuidelines()
	if err != nil {
		return err
	}
	err = a.generateLicense()
	if err != nil {
		return err
	}
	err = a.generateGitIgnore()
	if err != nil {
		return err
	}
	err = a.generatePullRequestTemplate()
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
	err = a.generateContinuousIntegration()
	if err != nil {
		return err
	}
	err = a.generateMakeFile()
	if err != nil {
		return err
	}
	err = a.generateDockerConcourseFile()
	if err != nil {
		return err
	}
	err = a.generateNomadFile()
	if err != nil {
		return err
	}
	err = a.generateMainFile()
	if err != nil {
		return err
	}
	err = a.generateConfigFiles()
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

	err = a.generateSwaggerSpec()
	if err != nil {
		return err
	}

	err = a.generateAPIFiles()
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

	err = a.generateHandlers()
	if err != nil {
		return err
	}

	err = a.generateRoutes()
	if err != nil {
		return err
	}

	err = a.generateMappers()
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

	err = a.generateEventFiles()
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateReadMe() error {
	fileToGen := fileGenerationVars{
		filePath:      "readme",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateContributionGuidelines() error {
	fileToGen := fileGenerationVars{
		filePath:      "contributing",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateLicense() error {
	fileToGen := fileGenerationVars{
		filePath:      "license",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateGitIgnore() error {
	fileToGen := fileGenerationVars{
		filePath:      ".gitignore",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generatePullRequestTemplate() error {
	fileToGen := fileGenerationVars{
		filePath:      ".github/PULL_REQUEST_TEMPLATE",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateIssuesFile() error {
	fileToGen := fileGenerationVars{
		filePath:      ".github/ISSUES_TEMPLATE",
		fileExtension: ".md",
	}
	err := a.generateFileFromTemplate(fileToGen)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateContinuousIntegration() error {
	tmplFilesToGen := []fileGenerationVars{
		{
			filePath:      "ci/build",
			fileExtension: ".yml",
		},
		{
			filePath:      "ci/unit",
			fileExtension: ".yml",
		},
		{
			filePath:      "ci/scripts/build",
			fileExtension: ".sh",
		},
		{
			filePath:      "ci/scripts/unit",
			fileExtension: ".sh",
		},
	}
	os.MkdirAll(a.pathToRepo+"ci/scripts", os.ModePerm)
	for _, fileToGen := range tmplFilesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a application) generateConfigFiles() error {
	return nil
}

func (a application) generateMainFile() error {
	return nil
}

func (a application) generateNomadFile() error {
	return nil
}

func (a application) generateDockerConcourseFile() error {
	return nil
}

func (a application) generateMakeFile() error {
	return nil
}

func (a application) generateAPIFiles() error {
	return nil
}

func (a application) generateSwaggerSpec() error {
	return nil
}

func (a application) generateMappers() error {
	tmplFilesToGen := []fileGenerationVars{
		{
			filePath:      "mapper/mapper",
			fileExtension: ".go",
		},
		{
			filePath:      "mapper/mapper_test",
			fileExtension: ".go",
		},
	}
	os.MkdirAll(a.pathToRepo+"mapper", os.ModePerm)
	for _, fileToGen := range tmplFilesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a application) generateFileFromTemplate(fileToGen fileGenerationVars) error {
	f, err := os.Create(a.pathToRepo + fileToGen.filePath + fileToGen.fileExtension)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	tmpl := template.Must(template.ParseFiles("./app-generation/content/templates/" + fileToGen.filePath + ".tmpl"))
	defer func() {
		writer.Flush()
		f.Close()
	}()
	err = tmpl.Execute(writer, a.templateVars)
	if err != nil {
		return err
	}
	return nil
}

func (a application) generateRoutes() error {
	tmplFilesToGen := []fileGenerationVars{
		{
			filePath:      "routes/routes",
			fileExtension: ".go",
		},
		{
			filePath:      "routes/routes_test",
			fileExtension: ".go",
		},
	}
	os.MkdirAll(a.pathToRepo+"routes", os.ModePerm)
	for _, fileToGen := range tmplFilesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a application) generateHandlers() error {
	tmplFilesToGen := []fileGenerationVars{
		{
			filePath:      "handlers/handlers",
			fileExtension: ".go",
		},
		{
			filePath:      "handlers/handlers_test",
			fileExtension: ".go",
		},
	}
	os.MkdirAll(a.pathToRepo+"handlers", os.ModePerm)
	for _, fileToGen := range tmplFilesToGen {
		err := a.generateFileFromTemplate(fileToGen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a application) generateEventFiles() error {

	return nil
}

func (a application) finaliseModules(pathToRepo string) error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
