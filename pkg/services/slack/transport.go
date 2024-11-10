package slack

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/database"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/slackapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
	contracts "github.com/ziplineeci/ziplinee-ci-contracts"
	crypt "github.com/ziplineeci/ziplinee-ci-crypt"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

// NewHandler returns a pubsub.Handler
func NewHandler(secretHelper crypt.SecretHelper, config *api.APIConfig, slackapiClient slackapi.Client, databaseClient database.Client, ziplineeService ziplinee.Service) Handler {
	return Handler{
		config:          config,
		secretHelper:    secretHelper,
		slackapiClient:  slackapiClient,
		databaseClient:  databaseClient,
		ziplineeService: ziplineeService,
	}
}

type Handler struct {
	config          *api.APIConfig
	secretHelper    crypt.SecretHelper
	slackapiClient  slackapi.Client
	databaseClient  database.Client
	ziplineeService ziplinee.Service
}

func (h *Handler) Handle(c *gin.Context) {

	// https://api.slack.com/slash-commands

	var slashCommand slackapi.SlashCommand
	// This will infer what binder to use depending on the content-type header.
	err := c.Bind(&slashCommand)
	if err != nil {
		log.Error().Err(err).Msg("Binding form data from Slack command webhook failed")
		c.String(http.StatusInternalServerError, "Binding form data from Slack command webhook failed")
		return
	}

	hasValidVerificationToken := h.hasValidVerificationToken(slashCommand)
	if !hasValidVerificationToken {
		log.Warn().Str("expectedToken", h.config.Integrations.Slack.AppVerificationToken).Str("actualToken", slashCommand.Token).Msg("Verification token for Slack command is invalid")
		c.String(http.StatusBadRequest, "Verification token for Slack command is invalid")
		return
	}

	if slashCommand.Command == "/ziplinee" {
		if slashCommand.Text != "" {
			splittedText := strings.Split(slashCommand.Text, " ")
			if len(splittedText) > 0 {
				command := splittedText[0]
				arguments := splittedText[1:]
				switch command {
				case "encrypt":

					log.Debug().Msg("Handling slash command /ziplinee encrypt")

					encryptedString, err := h.secretHelper.Encrypt(strings.Join(arguments, " "), "")
					if err != nil {
						log.Error().Err(err).Interface("slashCommand", slashCommand).Msg("Failed to encrypt secret")
						c.String(http.StatusOK, "Incorrect usage of /ziplinee encrypt")
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("ziplinee.secret(%v)", encryptedString))
					return

				case "release":

					log.Debug().Msg("Handling slash command /ziplinee release")

					// # release latest green version (for release branch?)
					// /ziplinee release github.com/ziplinee/ziplinee-ci-builder beta
					// /ziplinee release github.com/ziplinee/ziplinee-ci-api tooling

					// # release latest green version for particular branch
					// /ziplinee release github.com/ziplinee/ziplinee-ci-builder beta master

					// # release specific version
					// /ziplinee release github.com/ziplinee/ziplinee-ci-builder beta 0.0.47
					// /ziplinee release github.com/ziplinee/ziplinee-ci-api beta 0.0.130

					if len(arguments) < 3 {
						c.String(http.StatusOK, "You have to few arguments, the command has to be of type /ziplinee release <repo> <release> <version>")
						return
					}

					// retrieve pipeline with first argument
					fullRepoName := arguments[0]
					releaseName := arguments[1]
					buildVersion := arguments[2]

					fullRepoNameArray := strings.Split(fullRepoName, "/")
					if len(fullRepoNameArray) != 1 && len(fullRepoNameArray) != 3 {
						c.String(http.StatusOK, "Your repository needs to be of the form <repo name> or <repo source>/<repo owner>/<repo name>")
						return
					}

					var pipeline *contracts.Pipeline
					if len(fullRepoNameArray) == 1 {
						pipelines, err := h.databaseClient.GetPipelinesByRepoName(c.Request.Context(), fullRepoName, false)
						if err != nil {
							log.Error().Err(err).Msgf("Failed retrieving pipelines for repo name %v by name", fullRepoName)
							c.String(http.StatusOK, fmt.Sprintf("Retrieving the pipeline for repository %v from the database failed: %v", fullRepoName, err))
							return
						}
						if len(pipelines) <= 0 {
							c.String(http.StatusOK, fmt.Sprintf("The repo %v in your command does not have any ziplinee builds", fullRepoName))
							return
						}
						if len(pipelines) > 1 {
							commandsExample := ""
							for _, p := range pipelines {
								commandsExample += fmt.Sprintf("/ziplinee release %v/%v/%v %v %v\n", p.RepoSource, p.RepoOwner, p.RepoName, releaseName, buildVersion)
							}
							c.String(http.StatusOK, fmt.Sprintf("There are multiple pipelines with name %v, use the full name instead:\n%v", fullRepoName, commandsExample))
							return
						}
						pipeline = pipelines[0]
					} else {
						pipeline, err := h.databaseClient.GetPipeline(c.Request.Context(), fullRepoNameArray[0], fullRepoNameArray[1], fullRepoNameArray[2], map[api.FilterType][]string{}, false)
						if err != nil {
							c.String(http.StatusOK, fmt.Sprintf("Retrieving the pipeline for repository %v from the database failed: %v", fullRepoName, err))
							return
						}
						if pipeline == nil {
							c.String(http.StatusOK, fmt.Sprintf("The repo %v in your command does not have any ziplinee builds", fullRepoName))
							return
						}
					}

					// check if version exists
					builds, err := h.databaseClient.GetPipelineBuildsByVersion(c.Request.Context(), pipeline.RepoSource, pipeline.RepoOwner, pipeline.RepoName, buildVersion, []contracts.Status{contracts.StatusSucceeded}, 1, false)

					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Retrieving the build for repository %v and version %v from the database failed: %v", fullRepoName, buildVersion, err))
						return
					}

					var build *contracts.Build
					// get first build
					if len(builds) > 0 {
						build = builds[0]
					}
					if build == nil {
						c.String(http.StatusOK, fmt.Sprintf("The version %v in your command does not exist", buildVersion))
						return
					}
					if build.BuildStatus != contracts.StatusSucceeded {
						c.String(http.StatusOK, fmt.Sprintf("The build for version %v is not successful and cannot be used", buildVersion))
						return
					}

					// check if release target exists
					releaseExists := false
					for _, releaseTarget := range build.ReleaseTargets {
						if releaseTarget.Name == releaseName {
							// todo support release action

							releaseExists = true
							break
						}
					}

					if !releaseExists {
						c.String(http.StatusOK, fmt.Sprintf("The release %v in your command is not defined in the manifest", releaseName))
						return
					}

					// get user profile from api to set email address for TriggeredBy
					profile, err := h.slackapiClient.GetUserProfile(c.Request.Context(), slashCommand.UserID)
					if err != nil {
						c.String(http.StatusOK, fmt.Sprintf("Failed retrieving Slack user profile for user id %v: %v", slashCommand.UserID, err))
						return
					}

					// create release object and hand off to build service
					createdRelease, err := h.ziplineeService.CreateRelease(c.Request.Context(), contracts.Release{
						Name:           releaseName,
						Action:         "", // no support for releas action yet
						RepoSource:     build.RepoSource,
						RepoOwner:      build.RepoOwner,
						RepoName:       build.RepoName,
						ReleaseVersion: buildVersion,
						Groups:         build.Groups,
						Organizations:  build.Organizations,

						Events: []manifest.ZiplineeEvent{
							{
								Fired: true,
								Manual: &manifest.ZiplineeManualEvent{
									UserID: profile.Email,
								},
							},
						},
					}, *build.ManifestObject, build.RepoBranch, build.RepoRevision)

					if err != nil {
						errorMessage := fmt.Sprintf("Failed creating release %v for pipeline %v/%v/%v version %v for release command issued by %v", releaseName, build.RepoSource, build.RepoOwner, build.RepoName, buildVersion, profile.Email)
						log.Error().Err(err).Msg(errorMessage)
						c.String(http.StatusOK, fmt.Sprintf("Inserting starting the release: %v", err))
						return
					}

					c.String(http.StatusOK, fmt.Sprintf("Started releasing version %v to %v: %vpipelines/%v/%v/%v/releases/%v/logs", buildVersion, releaseName, h.config.APIServer.BaseURL, build.RepoSource, build.RepoOwner, build.RepoName, createdRelease.ID))
					return
				}
			}
		}
	}

	c.String(http.StatusOK, "Aye aye!")
}

func (h *Handler) hasValidVerificationToken(slashCommand slackapi.SlashCommand) bool {
	return slashCommand.Token == h.config.Integrations.Slack.AppVerificationToken
}
