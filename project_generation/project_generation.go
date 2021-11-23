package project_generation

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ONSdigital/log.go/v2/log"
)

type templateModel struct {
	Name        string
	Description string
	Year        int
	GoVersion   string
	Port        string
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
	templatePath string
	outputPath   string
	extension    string
	filePrefix   string
	executable   bool
}

type ProjectType string

const (
	GenericProject  ProjectType = "generic-project"
	BaseApplication ProjectType = "base-application"
	API             ProjectType = "api"
	Controller      ProjectType = "controller"
	EventDriven     ProjectType = "event-driven"
	Library         ProjectType = "library"
)

// To be replaced by `make install` with the user's own templates path
var templatesPath string = "/Users/USERNAME/dev/ons/dp/dp-cli/project_generation/content/templates"

// GenerateProject is the entry point into generating a project
func GenerateProject(appName, appDesc, projType, projectLocation, goVer, port string, repositoryCreated bool) error {
	ctx := context.Background()
	var err error

	an, ad, pt, pl, gv, prt, err := configureAndValidateArguments(ctx, appName, appDesc, projType, projectLocation, goVer, port)
	if err != nil {
		log.Error(ctx, "error configuring and validating arguments", err)
		return err
	}
	// If repository was created then this would have already been offered
	if !repositoryCreated {
		OfferPurgeProjectDestination(ctx, filepath.Join(pl, an))
	}

	newApp := application{
		pathToRepo:    filepath.Join(pl, an),
		projectType:   ProjectType(pt),
		name:          an,
		templateModel: PopulateTemplateModel(an, ad, gv, prt),
	}

	switch newApp.projectType {
	case GenericProject:
		err := newApp.generateGenericContent()
		if err != nil {
			return err
		}
	case BaseApplication:
		err := InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		if err != nil {
			return err
		}
		err = newApp.generateApplicationContent()
		if err != nil {
			return err
		}
		FinaliseModules(ctx, newApp.pathToRepo)
		FormatGoFiles(ctx, newApp.pathToRepo)
	case API:
		err := InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		if err != nil {
			return err
		}
		err = newApp.generateAPIContent()
		if err != nil {
			return err
		}
		FinaliseModules(ctx, newApp.pathToRepo)
		FormatGoFiles(ctx, newApp.pathToRepo)
	case Controller:
		err := InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		if err != nil {
			return err
		}
		err = newApp.generateControllerContent()
		if err != nil {
			return err
		}
		FinaliseModules(ctx, newApp.pathToRepo)
		FormatGoFiles(ctx, newApp.pathToRepo)
	case EventDriven:
		err := InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		if err != nil {
			return err
		}
		err = newApp.generateEventDrivenContent()
		if err != nil {
			return err
		}
		FinaliseModules(ctx, newApp.pathToRepo)
		FormatGoFiles(ctx, newApp.pathToRepo)
	case Library:
		err := InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		if err != nil {
			return err
		}
		err = newApp.generateLibraryContent()
		if err != nil {
			return err
		}
		FinaliseModules(ctx, newApp.pathToRepo)
		FormatGoFiles(ctx, newApp.pathToRepo)
	default:
		log.Error(ctx, "unable to generate project due to unknown project type given", err)
	}

	log.Info(ctx, "Project creation complete. Project can be found at "+newApp.pathToRepo)
	return nil
}

// createGenericContentDirectoryStructure will create child directories for Generic content at a given path
func (a application) createGenericContentDirectoryStructure() error {
	return os.MkdirAll(filepath.Join(a.pathToRepo, ".github"), os.ModePerm)
}

// createApplicationContentDirectoryStructure will create child directories for Application content at a given path
func (a application) createApplicationContentDirectoryStructure() error {
	os.MkdirAll(filepath.Join(a.pathToRepo, "config"), os.ModePerm)
	os.MkdirAll(filepath.Join(a.pathToRepo, "features/steps"), os.ModePerm)
	os.MkdirAll(filepath.Join(a.pathToRepo, "ci/scripts"), os.ModePerm)
	return nil
}

// createApplicationContentDirectoryStructure will create child directories for Application content at a given path
func (a application) createLibraryContentDirectoryStructure() error {
	os.MkdirAll(filepath.Join(a.pathToRepo, "ci/scripts"), os.ModePerm)
	return nil
}

// createAPIContentDirectoryStructure will create child directories for API content at a given path
func (a application) createAPIContentDirectoryStructure() error {
	err := os.MkdirAll(filepath.Join(a.pathToRepo, "api"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "service/mock"), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// createControllerContentDirectoryStructure will create child directories for Controller content at a given path
func (a application) createControllerContentDirectoryStructure() error {
	err := os.MkdirAll(filepath.Join(a.pathToRepo, "handlers"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "routes"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "mapper"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "service/mocks"), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// createEventDrivenContentDirectoryStructure will create child directories for Event Driven content at a given path
func (a application) createEventDrivenContentDirectoryStructure() error {
	err := os.MkdirAll(filepath.Join(a.pathToRepo, "cmd/producer"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "event/mock"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "schema"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "service/mock"), os.ModePerm)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(a.pathToRepo, "features/steps"), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// generateGenericContent will create all files for Generic content
func (a application) generateGenericContent() error {
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
	applyFilePrefixesToManifest(applicationFiles, a.name)
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

// generateLibraryContent will create all files for Application content
func (a application) generateLibraryContent() error {
	applyFilePrefixesToManifest(applicationFiles, a.name)
	err := a.generateGenericContent()
	if err != nil {
		return err
	}

	err = a.createLibraryContentDirectoryStructure()
	if err != nil {
		return err
	}

	err = a.generateBatchOfFileTemplates(libraryFiles)
	if err != nil {
		return err
	}

	return nil
}

func applyFilePrefixesToManifest(f []fileGen, prefix string) {
	for i := 0; i < len(f); i++ {
		if f[i].extension == ".nomad" {
			f[i].filePrefix = prefix
		}
	}
}

// generateAPIContent will create all files for API content
func (a application) generateAPIContent() error {
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
	outputFilename := fileToGen.filePrefix + fileToGen.outputPath + fileToGen.extension
	outputFilePath := filepath.Join(a.pathToRepo, outputFilename)
	f, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	tmpl := template.Must(template.ParseFiles(filepath.Join(templatesPath, fileToGen.templatePath+".tmpl")))

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

	if fileToGen.executable {
		err = os.Chmod(outputFilePath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
