package repository

import (
	"context"
	"fmt"
	"github.com/ONSdigital/log.go/log"
	"os/exec"
)

func CloneRepository(ctx context.Context, cloneUrl, projectLocation string) {
	// TODO prompt to blitz dir here or log and return (do nothing)
	fmt.Println("cloneUrl is: " + cloneUrl)
	cmd := exec.Command("git", "clone", cloneUrl)
	cmd.Dir = projectLocation
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during git clone", log.Error(err))
	}
}

func PushToRepo(ctx context.Context, projectLocation, appName string) {
	createNewBranch(ctx,projectLocation, appName)
	commitProject(ctx, projectLocation, appName)
	cmd := exec.Command("git", "push", "-u", "origin", "feature/boilerplate-generation")
	cmd.Dir = projectLocation+appName
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error during push", log.Error(err))
	}
}

func createNewBranch(ctx context.Context, projectLocation, appNme string)  {
	stageAllFiles(ctx, projectLocation, appNme)
	cmd := exec.Command("git", "checkout", "-b", "feature/boilerplate-generation")
	cmd.Dir = projectLocation + appNme
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error committing", log.Error(err))
	}
}

func commitProject(ctx context.Context, projectLocation, appNme string) {
	stageAllFiles(ctx, projectLocation, appNme)
	cmd := exec.Command("git", "commit", "-S", "-m", "initial commit, created via dp project generation tool")
	cmd.Dir = projectLocation + appNme
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error committing", log.Error(err))
	}
}

func stageAllFiles(ctx context.Context, projectLocation, appNme string) {
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = projectLocation + appNme
	err := cmd.Run()
	if err != nil {
		log.Event(ctx, "error staging files", log.Error(err))
	}
}
