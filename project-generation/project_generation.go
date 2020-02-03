package projectgeneration

import (
	"bufio"
	"context"
	"os"
	"text/template"

	"github.com/ONSdigital/log.go/log"
)

type templateModel struct {
	Name      string
	Year      int
	GoVersion string
	Port      string
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
}

type ProjectType string

const (
	GenericProject  ProjectType = "generic-project"
	BaseApplication ProjectType = "base-application"
	API             ProjectType = "api"
	Controller      ProjectType = "controller"
	EventDriven     ProjectType = "event-driven"
)

// GenerateProject is the entry point into generating a project
func GenerateProject(appName, projType, projectLocation, goVer, port string, repositoryCreated bool) error {
	ctx := context.Background()
	var err error

	an, pt, pl, gv, prt, err := configureAndValidateArguments(ctx, appName, projType, projectLocation, goVer, port)
	if err != nil {
		log.Event(ctx, "error configuring and validating arguments", log.Error(err))
		return err
	}
	// If repository was created then this would have already been offered
	if !repositoryCreated {
		OfferPurgeProjectDestination(ctx, pl, an)
	}

	newApp := application{
		pathToRepo:    pl + an + "/",
		projectType:   ProjectType(pt),
		name:          an,
		templateModel: PopulateTemplateModel(an, gv, prt),
	}

	switch newApp.projectType {
	case GenericProject:
		err := newApp.generateGenericContent()
		if err != nil {
			return err
		}
	case BaseApplication:
		InitGoModules(ctx, newApp.pathToRepo, newApp.name)
		err := newApp.generateApplicationContent()
		if err != nil {
			return err
		}
	case API:
		err := newApp.generateAPIContent()
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

	FinaliseModules(ctx, newApp.pathToRepo)

	log.Event(ctx, "Project creation complete. Project can be found at "+newApp.pathToRepo)
	return nil
}

// createGenericContentDirectoryStructure will create child directories for Generic content at a given path
func (a application) createGenericContentDirectoryStructure() error {
	return os.MkdirAll(a.pathToRepo+".github", os.ModePerm)
}

// createApplicationContentDirectoryStructure will create child directories for Application content at a given path
func (a application) createApplicationContentDirectoryStructure() error {
	os.MkdirAll(a.pathToRepo+"config", os.ModePerm)
	os.MkdirAll(a.pathToRepo+"ci/scripts", os.ModePerm)
	return nil
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

func applyFilePrefixesToManifest(f []fileGen, prefix string) {
	for i := 0; i < len(f); i++ {
		if f[i].templatePath == "nomad" {
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
	f, err := os.Create(a.pathToRepo + fileToGen.filePrefix + fileToGen.outputPath + fileToGen.extension)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(f)
	tmpl := template.Must(template.ParseFiles("./project-generation/content/templates/" + fileToGen.templatePath + ".tmpl"))

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
