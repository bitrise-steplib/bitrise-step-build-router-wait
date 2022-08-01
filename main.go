package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		if build.IsRunning() {
			log.Printf("- %s %s %s", build.TriggeredWorkflow, build.StatusText, buildURL)
		} else if build.IsSuccessful() {
			log.Donef("- %s successful %s)", build.TriggeredWorkflow, buildURL)
		} else if build.IsFailed() {
			log.Errorf("- %s failed", build.TriggeredWorkflow)
			failReason = "failed"
		} else if build.IsAborted() {
			log.Warnf("- %s aborted", build.TriggeredWorkflow)
			failReason = "aborted"
		} else if build.IsAbortedWithSuccess() {
			log.Infof("- %s cancelled", build.TriggeredWorkflow)
		}

		if cfg.AbortBuildsOnFail == "yes" && (build.IsAborted() || build.IsFailed()) {
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
		if build.IsRunning() == false {
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
