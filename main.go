package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// Config ...
type Config struct {
	AppSlug     string          `env:"BITRISE_APP_SLUG,required"`
	AccessToken stepconf.Secret `env:"access_token,required"`
	BuildSlugs  string          `env:"buildslugs,required"`
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

	app := bitrise.App{
		Slug:        cfg.AppSlug,
		AccessToken: string(cfg.AccessToken),
	}

	log.Infof("Waiting for builds:")

	buildSlugs := strings.Split(cfg.BuildSlugs, "\n")

	if err := app.WaitForBuilds(buildSlugs, func(build bitrise.Build) {
		switch build.Status {
		case 0:
			log.Printf("- %s %s", build.TriggeredWorkflow, build.StatusText)
		case 1:
			log.Donef("- %s successful", build.TriggeredWorkflow)
		case 2:
			log.Errorf("- %s failed", build.TriggeredWorkflow)
		case 3:
			log.Warnf("- %s aborted", build.TriggeredWorkflow)
		case 4:
			log.Infof("- %s cancelled", build.TriggeredWorkflow)
		}
	}); err != nil {
		failf("An error occoured: %s", err)
	}
}
