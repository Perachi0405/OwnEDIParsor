package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Perachi0405/ownEDIParsor/cli/cmd"
)

var (
	// To populate these vars from build/run
	//   go build/run -ldflags "-X main.gitCommit=$(git rev-parse HEAD) -X main.buildEpochSec=$(date +%s)" ...
	gitCommit     string
	buildEpochSec string
)

func main() {
	fmt.Print("in main")
	if err := cmd.Execute(getGitCommit(), getBuildEpochSec()); err != nil {
		os.Exit(1)
		fmt.Println("Errorin Main", err)
	}
	// serversamp = cmd.
	fmt.Println("ServerCommand")

}

func getGitCommit() string {
	fmt.Println("Inside the getGitCommit()")
	shaPrefix := func(sha string) string {
		return string(([]rune(sha))[:7])
	}
	fmt.Println("getGitCommit() shaPrefix")
	if gitCommit != "" {
		gitCommit = shaPrefix(gitCommit)
		return gitCommit
	}
	// https://devcenter.heroku.com/articles/dyno-metadata
	gitCommit = os.Getenv("HEROKU_SLUG_COMMIT")
	if gitCommit != "" {
		gitCommit = shaPrefix(gitCommit)
		return gitCommit
	}
	// but sometimes, HEROKU_SLUG_COMMIT is might be empty, let's
	// try HEROKU_SLUG_DESCRIPTION, which seems more reliable.
	gitCommit = os.Getenv("HEROKU_SLUG_DESCRIPTION")
	// although HEROKU_SLUG_DESCRIPTION is of "Deploy <sha>" format
	if strings.HasPrefix(gitCommit, "Deploy ") {
		gitCommit = shaPrefix(gitCommit[len("Deploy "):])
		return gitCommit
	}
	gitCommit = "(unknown)"
	fmt.Print("in git commit", gitCommit)
	return gitCommit
}

func getBuildEpochSec() string {
	fmt.Println("getBuildEpochSec() Function")
	if buildEpochSec != "" {
		fmt.Print("in git buildEpochSec", buildEpochSec)
		return buildEpochSec
	}
	buildEpochSec = "(unknown)"
	fmt.Print("in git buildEpochSec", buildEpochSec)
	return buildEpochSec
}
