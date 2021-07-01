package main

import (
	"fmt"
	"os"
	"strings"
	"path/filepath"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/bitrise-step-build-router-start/bitrise"
)

// Config ...
type Config struct {
	AppSlug                string          `env:"BITRISE_APP_SLUG,required"`
	AccessToken            stepconf.Secret `env:"access_token,required"`
	BuildSlugs             string          `env:"buildslugs,required"`
	BuildArtifactsSavePath string          `env:"build_artifacts_save_path"`
	AbortBuildsOnFail      string          `env:"abort_on_fail"`
	IsVerboseLog           bool            `env:"verbose,required"`
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
		var failReason string
		var buildURL = fmt.Sprintf("(https://app.bitrise.io/build/%s)", build.Slug)
		switch build.Status {
		case 0:
			log.Printf("- %s %s %s", build.TriggeredWorkflow, build.StatusText, buildURL)
		case 1:
			log.Donef("- %s successful %s)", build.TriggeredWorkflow, buildURL)
		case 2:
			log.Errorf("- %s failed", build.TriggeredWorkflow)
			failReason = "failed"
		case 3:
			log.Warnf("- %s aborted", build.TriggeredWorkflow)
			failReason = "aborted"
		case 4:
			log.Infof("- %s cancelled", build.TriggeredWorkflow)
			failReason = "cancelled"
		}

		if cfg.AbortBuildsOnFail == "yes" && build.Status > 1 {
			for _, buildSlug := range buildSlugs {
				if buildSlug != build.Slug {
					abortErr := app.AbortBuild(buildSlug, "Abort on Fail - Build [https://app.bitrise.io/build/"+build.Slug+"] "+failReason+"\nAuto aborted by parent build")
					if abortErr != nil {
						log.Warnf("failed to abort build, error: %s", abortErr)
					}
					log.Donef("Build " + buildSlug + " aborted due to associated build failure")
				}
			}
		}
		if build.Status != 0 {
			buildArtifactSaveDir := strings.TrimSpace(cfg.BuildArtifactsSavePath)
			if buildArtifactSaveDir != "" {
				artifactsResponse, err := build.GetBuildArtifacts(app)
				if err != nil {
					log.Warnf("failed to get build artifacts, error: %s", err)
				}
				for _, artifactSlug := range artifactsResponse.ArtifactSlugs {
					artifactObj, err := build.GetBuildArtifact(app, artifactSlug.ArtifactSlug)
					if err != nil {
						log.Warnf("failed to get build artifact, error: %s", err)
					}

					fullBuildArtifactsSavePath := filepath.Join(buildArtifactSaveDir, artifactObj.Artifact.Title)
					downloadErr := artifactObj.Artifact.DownloadArtifact(fullBuildArtifactsSavePath)
					if downloadErr != nil {
						log.Warnf("failed to download artifact, error: %s", downloadErr)
					}
					log.Donef("Downloaded: " + artifactObj.Artifact.Title + " to path " + strings.TrimSpace(fullBuildArtifactsSavePath))
				}
			}
		}
	}); err != nil {
		failf("An error occurred: %s", err)
	}
}
