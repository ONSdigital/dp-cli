package repository

import (
	"context"
	"github.com/ONSdigital/log.go/log"
	"os/exec"
)

// CloneRepository will clone a given repository at a given location,
// the location is the projectLocation joined with appName
func CloneRepository(ctx context.Context, cloneUrl, projectLocation, appName string) error {
	cmd := exec.Command("git", "clone", cloneUrl)
	cmd.Dir = projectLocation
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during git clone", log.Error(err))
		return err
	}
	err = switchRepoToSSH(ctx, projectLocation, appName)
	if err != nil {
		log.Event(ctx, "failed to switch repo to SSH", log.Error(err))
		return err
	}
	return nil
}

// PushToRepo will push the contents of a given local directory to a set project remote (.git)
func PushToRepo(ctx context.Context, projectLocation, appName string) error {
	err := createBoilerPlateBranch(ctx, projectLocation, appName)
	if err != nil {
		return err
	}
	err = commitProject(ctx, projectLocation, appName)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "push", "-u", "origin", "feature/boilerplate-generation")
	cmd.Dir = projectLocation + appName
	err = cmd.Run()
	if err != nil {
		log.Event(ctx, "error during push", log.Error(err))
		return err
	}
	return nil
}

// switchRepoToSSH will convert a given locations repositories connection from HTTPS to SSH
func switchRepoToSSH(ctx context.Context, projectLocation, appName string) error {
	cmd := exec.Command("git", "remote", "set-url", "origin", "git@github.com:"+org+"/"+appName+".git")
	cmd.Dir = projectLocation + appName
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "switching origin access protocols from HTTPS to SSH", log.Error(err))
		return err
	}
	return nil
}

// createBoilerPlateBranch will create a new branch named "feature/boilerplate-generation" locally
func createBoilerPlateBranch(ctx context.Context, projectLocation, appNme string) error {
	err := stageAllFiles(ctx, projectLocation, appNme)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "checkout", "-b", "feature/boilerplate-generation")
	cmd.Dir = projectLocation + appNme
	err = cmd.Run()
	if err != nil {
		log.Event(ctx, "error committing", log.Error(err))
		return err
	}
	return nil
}

// commitProject will stage all files in a given directory and then commit
// them to the current working branch on source control
func commitProject(ctx context.Context, projectLocation, appNme string) error {
	err := stageAllFiles(ctx, projectLocation, appNme)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "commit", "-S", "-m", "initial commit, created via dp project generation tool")
	cmd.Dir = projectLocation + appNme
	err = cmd.Run()
	if err != nil {
		log.Event(ctx, "error committing", log.Error(err))
		return err
	}
	return nil
}

// stageAllFiles will stage all files at a given directory
func stageAllFiles(ctx context.Context, projectLocation, appNme string) error {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = projectLocation + appNme
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error staging files", log.Error(err))
		return err
	}
	return nil
}
