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
	"time"
)

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
		err = OfferPurgeProjDestination(ctx, validatedProjectLocation, appName)
		if err != nil {
			return validatedProjectLocation, err
		}
	}
	return validatedProjectLocation, err
}

// OfferPurgeProjDestination will offer the user an option to purge the contents at a given location
func OfferPurgeProjDestination(ctx context.Context, projectLoc, appName string) error {
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
