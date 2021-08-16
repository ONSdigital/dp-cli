package project_generation

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ONSdigital/log.go/log"
)

type ListOfArguments map[string]*Argument
type Argument struct {
	Validator func(context.Context, string) (string, error)
	Context   context.Context
	InputVal  string
	OutputVal string
}

func configureAndValidateArguments(ctx context.Context, appName, appDesc, projectType, projectLocation, goVersion, port string) (an, ad, pt, pl, gv, prt string, err error) {
	listOfArguments := make(ListOfArguments)
	listOfArguments["appName"] = &Argument{
		InputVal:  appName,
		Context:   ctx,
		Validator: ValidateAppName,
	}
	listOfArguments["description"] = &Argument{
		InputVal:  appDesc,
		Context:   ctx,
		Validator: ValidateAppDescription,
	}
	listOfArguments["projectType"] = &Argument{
		InputVal:  projectType,
		Context:   ctx,
		Validator: ValidateProjectType,
	}
	listOfArguments["projectLocation"] = &Argument{
		InputVal:  projectLocation,
		Context:   ctx,
		Validator: ValidateProjectLocation,
	}
	listOfArguments, err = ValidateArguments(listOfArguments)
	if err != nil {
		log.Event(ctx, "validation error", log.Error(err))
		return "", "", "", "", "", "", err
	}
	an = listOfArguments["appName"].OutputVal
	ad = listOfArguments["description"].OutputVal
	pt = listOfArguments["projectType"].OutputVal
	pl = listOfArguments["projectLocation"].OutputVal
	listOfArguments = make(ListOfArguments)
	goVerUnset := goVersion == ""
	if goVerUnset && ProjectType(pt) != GenericProject {
		listOfArguments["goVersion"] = &Argument{
			InputVal:  goVersion,
			Context:   ctx,
			Validator: ValidateGoVersion,
		}
	} else {
		gv = goVersion
	}

	listOfArguments["port"] = &Argument{
		InputVal:  port,
		Context:   ctx,
		Validator: ValidatePortNumber,
	}

	listOfArguments, err = ValidateArguments(listOfArguments)
	if err != nil {
		log.Event(ctx, "validation error", log.Error(err))
		return "", "", "", "", "", "", err
	}
	prt = listOfArguments["port"].OutputVal
	if goVerUnset && ProjectType(pt) != GenericProject {
		gv = listOfArguments["goVersion"].OutputVal
	}
	return an, ad, pt, pl, gv, prt, nil
}

func ValidateArguments(arguments map[string]*Argument) (map[string]*Argument, error) {
	var err error = nil
	for key, value := range arguments {
		arguments[key].OutputVal, err = value.Validator(value.Context, value.InputVal)
		if err != nil {
			log.Event(context.Background(), "validation error ", log.Error(err))
			return nil, err
		}
	}

	return arguments, err
}

// ValidateAppName will ensure that the app name has been provided and is acceptable, if not it will keep
// prompting until it is
func ValidateAppName(ctx context.Context, name string) (string, error) {
	var err error = nil

	for name == "" {
		name, err = PromptForInput(ctx, "Please specify the name of the application, if this is a "+
			"Digital publishing specific application it should be prepended with 'dp-'")
		if err != nil {
			return "", err
		}
	}
	return name, err
}

// ValidateAppDescription will ensure that the app description has been provided and is acceptable, if not it will keep
// prompting until it is
func ValidateAppDescription(ctx context.Context, description string) (string, error) {
	var err error = nil

	for description == "" {
		description, err = PromptForInput(ctx, "Please specify a short description of the application:")
		if err != nil {
			return "", err
		}
	}
	return description, err
}

// ValidateProjectType will ensure that the project type provided by the users is one that can be boilerplate
func ValidateProjectType(ctx context.Context, projectType string) (validatedProjectType string, err error) {
	if projectType == "" {
		prompt := "Please specify the project type"
		options := []string{"generic-project", "base-application", "api", "controller", "event-driven", "library"}
		projectType, err = OptionPromptInput(ctx, prompt, options...)
		if err != nil {
			return "", err
		}
	}
	return projectType, err
}

//ValidateGoVersion will ensure that the golang docker hub image version provided by the user is valid
func ValidateGoVersion(ctx context.Context, goVer string) (string, error) {
	var err error = nil
	if ValidVersionNumber(goVer) {
		return goVer, nil
	}
	for !ValidVersionNumber(goVer) {
		goVer, err = PromptForInput(ctx, "Please specify the docker hub image version of GO to use:e.g.(1.x.x)")
		if err != nil {
			return "", err
		}
	}
	return goVer, nil
}

func ValidatePortNumber(ctx context.Context, port string) (validatedPort string, err error) {
	if port == "" {
		port, err = PromptForInput(ctx, "Please specify the port number for this app, or leave blank for a library:")
		if err != nil {
			return "", err
		}
	}
	return port, nil
}

// ValidateProjectLocation will ensure that the projects location has been provided and is acceptable.
// It will ensure the directory exists and has the option to offer a purge of files at that location
func ValidateProjectLocation(ctx context.Context, projectLocation string) (string, error) {
	var err error = nil

	for projectLocation == "" {
		projectLocation, err = PromptForInput(ctx, "Please specify a directory for the project to be created in:")
		if err != nil {
			return "", err
		}
	}

	return projectLocation, nil
}

func ValidateProjectDirectory(ctx context.Context, path, projectName string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File path to project does not exists
		log.Event(ctx, "file path to project location does not exists - for safety assuming wrong location was provided")
		return err
	}
	projectPath := filepath.Join(path, projectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		// File path to project does exists but project directory does not exist at the given path
		err := os.Mkdir(projectPath, os.ModeDir|os.ModePerm)
		if err != nil {
			log.Event(ctx, "error creating project directory", log.Error(err))
			return err
		}
		return nil
	}
	// File path to project does exists and there is a project with the given name already present
	isEmptyDir, err := IsEmptyDir(projectPath)
	if err != nil {
		log.Event(ctx, "error checking if directory is empty", log.Error(err))
		return err
	}
	if !isEmptyDir {
		// Project directory exists at the given file path and has content inside of it
		err = OfferPurgeProjectDestination(ctx, projectPath)
		if err != nil {
			log.Event(ctx, "error during offer purge of directory", log.Error(err))
			return err
		}
	}
	//everything is good and nothing needs to be done
	return nil
}

// ValidateBranchingStrategy will ensure that the strategy  provided by the user is one that can be boilerplate
func ValidateBranchingStrategy(ctx context.Context, branchingStrategy string) (string, error) {
	if branchingStrategy == "" {
		prompt := "Please pick the branching strategy you wish this repo to use:"
		options := []string{"github flow", "git flow"}
		branchingStrategy, err := OptionPromptInput(ctx, prompt, options...)
		if err != nil {
			return "", err
		}
		branchingStrategy = strings.Replace(branchingStrategy, " flow", "", -1)
	}
	return branchingStrategy, nil
}

// OfferPurgeProjectDestination will offer the user an option to purge the contents at a given location
func OfferPurgeProjectDestination(ctx context.Context, projectPath string) error {
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		err = os.MkdirAll(projectPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// If path has files in then purge them... but check with user first (prompt are you sure)
	isEmpty, err := IsEmptyDir(projectPath)
	if err != nil {
		return err
	}

	if !isEmpty {
		//prompt user
		maxUserInputAttempts := 3
		deleteContents := PromptForConfirmation(ctx, "The directory "+projectPath+" was not empty would you "+
			"like to purge its contents, this will also remove any git files if present?", maxUserInputAttempts)

		if deleteContents {
			err := os.RemoveAll(projectPath)
			if err != nil {
				return err
			}
			err = os.MkdirAll(projectPath, os.ModePerm)
			if err != nil {
				return err
			}
		}
		fmt.Println("Path to generate files created")
	}
	return nil
}

// IsEmptyDir will check if a given directory is empty or not
func IsEmptyDir(path string) (isEmptyDir bool, err error) {
	f, err := os.Open(path)
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

// PopulateTemplateModel will populate the templating model with variables that can be used in templates
func PopulateTemplateModel(name, desc, goVer, port string) templateModel {
	// UTC to avoid any sketchy BST timing
	year := time.Now().UTC().Year()
	return templateModel{
		Name:        name,
		Description: desc,
		Year:        year,
		GoVersion:   goVer,
		Port:        port,
	}
}

// PromptForConfirmation will prompt for yes/no style answers on command-line
func PromptForConfirmation(ctx context.Context, prompt string, maxInputAttempts int) bool {
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

// PromptForInput will write a line to output then wait for input which is returned from the function
func PromptForInput(ctx context.Context, prompt string) (string, error) {
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(prompt)
	scanner.Scan()
	input = scanner.Text()
	if scanner.Err() != nil {
		log.Event(ctx, "Failed to read user input", log.Error(scanner.Err()))
		return "", scanner.Err()
	}
	return input, nil
}

func OptionPromptInput(ctx context.Context, prompt string, options ...string) (string, error) {
	var input string
	var optionSelected int
	scanner := bufio.NewScanner(os.Stdin)
	valid := false
	for !valid {
		fmt.Println(prompt + "\n")
		for i, option := range options {
			fmt.Printf("[%d] %v \n", i, option)
		}
		fmt.Println("\nPlease enter the number corresponding to your choice:")
		scanner.Scan()
		input = scanner.Text()
		if scanner.Err() != nil {
			log.Event(ctx, "failed to read user input", log.Error(scanner.Err()))
			return "", scanner.Err()
		}
		// If user entered the text rather than number, it will be accepted
		for _, opt := range options {
			if input == opt {
				return opt, nil
			}
		}

		optionSelected, err := strconv.Atoi(input)

		if err != nil || optionSelected > len(options)-1 || optionSelected < 0 {
			fmt.Println("\n selected option is not valid, please select from the range provided")
		} else {
			return options[optionSelected], nil
		}
	}
	return options[optionSelected], nil
}

// InitGoModules will initialise the go modules for a project at a given directory unless go.mod already exists
func InitGoModules(ctx context.Context, pathToRepo, name string) error {
	_, err := os.Stat(pathToRepo + "/go.mod")
	if os.IsExist(err) {
		return err // file already exists but there's some other error with it
	}
	if err == nil {
		return nil // file already exists, do nothing
	}

	cmd := exec.Command("go", "mod", "init", "github.com/ONSdigital/"+name)
	cmd.Dir = pathToRepo
	err = cmd.Run()
	if err != nil {
		log.Event(ctx, "error initialising go modules", log.Error(err))
	}
	return nil
}

// runGoModTidy will download all the dependencies that are required for your source file and updates go mod with
// that dependency.
func runGoModTidy(ctx context.Context, pathToRepo string) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error initialising go modules", log.Error(err))
	}
}

// FinaliseModules will run go build ./... to generate go modules dependency management files
func FinaliseModules(ctx context.Context, pathToRepo string) {
	runGoModTidy(ctx, pathToRepo)
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during go build step", log.Error(err))
	}
}

// FormatGoFiles will run go fmt ./... to ensure all generated code conforms to standards
func FormatGoFiles(ctx context.Context, pathToRepo string) {
	cmd := exec.Command("go", "fmt", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during go fmt step", log.Error(err))
	}
}

func ValidVersionNumber(ver string) bool {
	// Accept formats such as '0.1'and '9.7.15'
	var rxPat = regexp.MustCompile(`^([0-9]+\.){1,2}[0-9]+$`)

	return rxPat.MatchString(ver)
}
