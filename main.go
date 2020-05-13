package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
)

// Config ...
type Config struct {
	AppSlug      string          `env:"BITRISE_APP_SLUG,required"`
	AccessToken  stepconf.Secret `env:"access_token,required"`
	BuildSlugs   string          `env:"buildslugs,required"`
	IsVerboseLog bool            `env:"verbose,required"`
}

func failf(s string, a ...interface{}) {
	log.Errorf(s, a...)
	os.Exit(1)
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with an input: %s", err)
	}

	stepconf.Print(cfg)
	fmt.Println()

	log.SetEnableDebugLog(cfg.IsVerboseLog)

	app := bitrise.NewAppWithDefaultURL(cfg.AppSlug, string(cfg.AccessToken))

	log.Infof("Waiting for builds:")

	buildSlugs := strings.Split(cfg.BuildSlugs, "\n")

	if err := app.WaitForBuilds(buildSlugs, func(build bitrise.Build) {
		var buildURL = fmt.Sprintf("(https://app.bitrise.io/build/%s)", build.Slug)
		switch build.Status {
		case 0:
			log.Printf("- %s %s %s", build.TriggeredWorkflow, build.StatusText, buildURL)
		case 1:
			log.Donef("- %s successful %s)", build.TriggeredWorkflow, buildURL)
		case 2:
			log.Errorf("- %s failed %s", build.TriggeredWorkflow, buildURL)
		case 3:
			log.Warnf("- %s aborted %s", build.TriggeredWorkflow, buildURL)
		case 4:
			log.Infof("- %s cancelled %s", build.TriggeredWorkflow, buildURL)
		}
	}); err != nil {
		failf("An error occurred: %s", err)
	}
}
