package main

import (
	"log"
	"os"
	"strings"

	"github.com/Perachi0405/ownediparse/cli/cmd"
)

var (
	// To populate these vars from build/run
	//   go build/run -ldflags "-X main.gitCommit=$(git rev-parse HEAD) -X main.buildEpochSec=$(date +%s)" ...
	gitCommit     string
	buildEpochSec string
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	if err := cmd.Execute(getGitCommit(), getBuildEpochSec()); err != nil {
		os.Exit(1)
	}
	log.Println("Checking main")
}

func getGitCommit() string {
	shaPrefix := func(sha string) string {
		return string(([]rune(sha))[:7])
	}
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
	return gitCommit
}

func getBuildEpochSec() string {
	if buildEpochSec != "" {
		return buildEpochSec
	}
	buildEpochSec = "(unknown)"
	return buildEpochSec
}
