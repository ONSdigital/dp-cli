package projectgeneration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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

func configureAndValidateArguments(ctx context.Context, appName, projectType, projectLocation, goVersion, port string) (an, pt, pl, gv, prt string, err error) {
	listOfArguments := make(ListOfArguments)
	listOfArguments["appName"] = &Argument{
		InputVal:  appName,
		Context:   ctx,
		Validator: ValidateAppName,
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
		return "", "", "", "", "", err
	}
	an = listOfArguments["appName"].OutputVal
	pt = listOfArguments["projectType"].OutputVal
	pl = listOfArguments["projectLocation"].OutputVal
	listOfArguments = make(ListOfArguments)
	goVerUnset := goVersion == "unset" || goVersion == ""
	if goVerUnset && projectType != "generic-project" {
		listOfArguments["goVersion"] = &Argument{
			InputVal:  goVersion,
			Context:   ctx,
			Validator: ValidateGoVersion,
		}
		gv = listOfArguments["goVersion"].OutputVal
	}

	if port == "" {
		listOfArguments["port"] = &Argument{
			InputVal:  port,
			Context:   ctx,
			Validator: ValidatePortNumber,
		}
		prt = listOfArguments["port"].OutputVal
	} else {
		prt = port
	}

	listOfArguments, err = ValidateArguments(listOfArguments)
	if err != nil {
		log.Event(ctx, "validation error", log.Error(err))
		return "", "", "", "", "", err
	}

	return an, pt, pl, gv, prt, nil
}

func ValidateArguments(arguments map[string]*Argument) (validArguments map[string]*Argument, err error) {
	fmt.Println("function ValidateArguments hit")
	validArguments = make(ListOfArguments)
	for key, value := range arguments {
		validArguments[key] = value
	}

	for key, value := range arguments {
		fmt.Println("key " + key)
		fmt.Println("value " + value.InputVal)
		validArguments[key].OutputVal, err = value.Validator(value.Context, value.InputVal)
		if err != nil {
			log.Event(context.Background(), "validation error ", log.Error(err))
			return nil, err
		}
	}

	return validArguments, err
}

// ValidateAppName will ensure that the app name has been provided and is acceptable, if not it will keep
// prompting until it is
func ValidateAppName(ctx context.Context, unvalidatedAppName string) (validatedAppName string, err error) {
	if unvalidatedAppName == "unset" || unvalidatedAppName == "" {
		validatedAppName, err = PromptForInput(ctx, "Please specify the name of the application, if this is a "+
			"Digital publishing specific application it should be prepended with 'dp-'")
		if err != nil {
			return "", err
		}
	} else {
		validatedAppName = unvalidatedAppName
	}
	return validatedAppName, err
}

// ValidateProjectType will ensure that the project type provided by the users is one that can be boilerplate
func ValidateProjectType(ctx context.Context, projectType string) (validatedProjectType string, err error) {
	if projectType == "" || projectType == "unset" {
		prompt := "Please specify the project type"
		options := []string{"generic-project", "base-application", "api", "controller", "event-driven"}
		projectType, err = OptionPromptInput(ctx, prompt, options...)
		if err != nil {
			return "", err
		}
	}
	return projectType, err
}

func ValidateGoVersion(ctx context.Context, unvalidatedGoVersion string) (validatedGoVersion string, err error) {
	if unvalidatedGoVersion == "unset" || unvalidatedGoVersion == "" {
		validatedGoVersion, err = PromptForInput(ctx, "Please specify the version of GO to use")
		if err != nil {
			return "", err
		}
	} else {
		validatedGoVersion = unvalidatedGoVersion
	}
	return validatedGoVersion, err
}

func ValidatePortNumber(ctx context.Context, unvalidatedPort string) (validatedPort string, err error) {
	if unvalidatedPort == "" {
		validatedPort, err = PromptForInput(ctx, "Please specify the port number for this app")
		if err != nil {
			return "", err
		}
	} else {
		validatedPort = unvalidatedPort
	}
	return validatedPort, err
}

// ValidateProjectLocation will ensure that the projects location has been provided and is acceptable.
// It will ensure the directory exists and has the option to offer a purge of files at that location
func ValidateProjectLocation(ctx context.Context, unvalidatedProjectLocation string) (validatedProjectLocation string, err error) {
	if unvalidatedProjectLocation == "unset" || unvalidatedProjectLocation == "" {
		validatedProjectLocation, err = PromptForInput(ctx, "Please specify a directory for the project to be created in")
		if err != nil {
			return "", err
		}
	} else {
		validatedProjectLocation = unvalidatedProjectLocation
	}
	if validatedProjectLocation[len(validatedProjectLocation)-1:] != "/" {
		validatedProjectLocation = validatedProjectLocation + "/"
	}
	fmt.Println("validatedProjectLocation" + validatedProjectLocation)
	return validatedProjectLocation, err
}

func ValidateProjectDirectory(ctx context.Context, path, projectName string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File path to project does not exists
		log.Event(ctx, "file path to project location does not exists - for safety assuming wrong location was provided")
	} else {
		// File path to project does exists
		if _, err := os.Stat(path + projectName); os.IsNotExist(err) {
			// File path to project does exists but project directory does not exist at the given path
			err := os.Mkdir(path+projectName, os.ModeDir)
			if err != nil {
				log.Event(ctx, "error creating project directory", log.Error(err))
				return err
			}
		} else {
			// File path to project does exists and there is a project with the given name already present
			isEmptyDir, err := IsEmptyDir(path + projectName)
			if err != nil {
				log.Event(ctx, "error checking if directory is empty", log.Error(err))
				return err
			}
			if !isEmptyDir {
				// Project directory exists at the given file path and has content inside of it
				err = OfferPurgeProjectDestination(ctx, path, projectName)
				if err != nil {
					log.Event(ctx, "error during offer purge of directory", log.Error(err))
					return err
				}
			} //else everything is good and nothing needs to be done
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// OfferPurgeProjectDestination will offer the user an option to purge the contents at a given location
func OfferPurgeProjectDestination(ctx context.Context, projectLoc, appName string) error {
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
		deleteContents := PromptForConfirmation(ctx, "The directory "+projectLoc+appName+" was not empty would you "+
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
func PopulateTemplateModel(name, goVer, port string) templateModel {
	// UTC to avoid any sketchy BST timing
	year := time.Now().UTC().Year()
	return templateModel{
		Name:      name,
		Year:      year,
		GoVersion: goVer,
		Port:      port,
	}
}

// PromptForConfirmation will prompt for yes/no style answers on command line
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
	optionSelected, err := strconv.Atoi(input)
	if scanner.Err() != nil {
		log.Event(ctx, "failed to convert user input to valid option", log.Error(err))
		return "", scanner.Err()
	}
	return options[optionSelected], nil
}

// InitGoModules will initialise the go modules for a project at a given directory
func InitGoModules(ctx context.Context, pathToRepo, name string) {
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error initialising go modules", log.Error(err))
	}
}

// FinaliseModules will run go build ./... to generate go modules dependency management files
func FinaliseModules(ctx context.Context, pathToRepo string) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = pathToRepo
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during go build step", log.Error(err))
	}
}
