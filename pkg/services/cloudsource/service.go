package cloudsource

import (
	"context"
	"errors"

	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/cloudsourceapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/pubsubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/queue"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
	contracts "github.com/ziplineeci/ziplinee-ci-contracts"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

var (
	ErrNonCloneableEvent = errors.New("The event is not cloneable")
	ErrNoManifest        = errors.New("The repository has no manifest at the pushed commit")
)

// Service handles pubsub events for Cloud Source Repository integration
//
//go:generate mockgen -package=cloudsource -destination ./mock.go -source=service.go
type Service interface {
	CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error)
	IsAllowedProject(ctx context.Context, notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization)
}

// NewService returns a new bitbucket.Service
func NewService(config *api.APIConfig, cloudsourceapiClient cloudsourceapi.Client, pubsubapiClient pubsubapi.Client, ziplineeService ziplinee.Service, queueService queue.Service) Service {
	return &service{
		config:               config,
		cloudsourceapiClient: cloudsourceapiClient,
		pubsubapiClient:      pubsubapiClient,
		ziplineeService:      ziplineeService,
		queueService:         queueService,
	}
}

type service struct {
	config               *api.APIConfig
	cloudsourceapiClient cloudsourceapi.Client
	pubsubapiClient      pubsubapi.Client
	ziplineeService      ziplinee.Service
	queueService         queue.Service
}

func (s *service) CreateJobForCloudSourcePush(ctx context.Context, notification cloudsourceapi.PubSubNotification) (err error) {

	// check to see that it's a cloneable event

	if notification.RefUpdateEvent == nil {
		return ErrNonCloneableEvent
	}

	var commits []contracts.GitCommit
	var repoBranch string
	var repoRevision string
	for _, refUpdate := range notification.RefUpdateEvent.RefUpdates {
		commits = append(commits, contracts.GitCommit{
			Author: contracts.GitAuthor{
				Email:    notification.RefUpdateEvent.Email,
				Name:     notification.RefUpdateEvent.GetAuthorName(),
				Username: notification.RefUpdateEvent.GetAuthorName(),
			},
			Message: refUpdate.NewId,
		})
		repoBranch = refUpdate.GetRepoBranch()
		repoRevision = refUpdate.NewId
	}

	gitEvent := manifest.ZiplineeGitEvent{
		Event:      "push",
		Repository: notification.GetRepository(),
		Branch:     repoBranch,
	}

	// handle git triggers
	err = s.queueService.PublishGitEvent(ctx, gitEvent)
	if err != nil {
		return
	}

	// get access token
	accessToken, err := s.cloudsourceapiClient.GetAccessToken(ctx)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Access Token failed")
		return err
	}

	// get manifest file
	manifestExists, manifestString, err := s.cloudsourceapiClient.GetZiplineeManifest(ctx, accessToken, notification, nil)
	if err != nil {
		log.Error().Err(err).
			Msg("Retrieving Estafettte manifest failed")
		return err
	}

	if !manifestExists {
		return ErrNoManifest
	}

	// get organizations linked to integration
	_, organizations := s.IsAllowedProject(ctx, notification)

	// create build object and hand off to build service
	_, err = s.ziplineeService.CreateBuild(ctx, contracts.Build{
		RepoSource:    notification.GetRepoSource(),
		RepoOwner:     notification.GetRepoOwner(),
		RepoName:      notification.GetRepoName(),
		RepoBranch:    repoBranch,
		RepoRevision:  repoRevision,
		Manifest:      manifestString,
		Commits:       commits,
		Organizations: organizations,
		Events: []manifest.ZiplineeEvent{
			{
				Fired: true,
				Git:   &gitEvent,
			},
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed creating build for pipeline %v/%v/%v with revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)
		return err
	}

	log.Debug().Msgf("Created build for pipeline %v/%v/%v with revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)

	go func() {
		// create new context to avoid cancellation impacting execution
		span, _ := opentracing.StartSpanFromContext(ctx, "cloudsource:AsyncSubscribeToPubsubTriggers")
		ctx = opentracing.ContextWithSpan(context.Background(), span)
		defer span.Finish()

		err := s.pubsubapiClient.SubscribeToPubsubTriggers(ctx, manifestString)
		if err != nil {
			log.Error().Err(err).Msgf("Failed subscribing to topics for pubsub triggers for build %v/%v/%v revision %v", notification.GetRepoSource(), notification.GetRepoOwner(), notification.GetRepoName(), repoRevision)
		}
	}()

	return nil
}

func (s *service) IsAllowedProject(ctx context.Context, notification cloudsourceapi.PubSubNotification) (isAllowed bool, organizations []*contracts.Organization) {

	if len(s.config.Integrations.CloudSource.ProjectOrganizations) == 0 {
		return true, []*contracts.Organization{}
	}

	for _, po := range s.config.Integrations.CloudSource.ProjectOrganizations {
		if po.Project == notification.GetRepoOwner() {
			return true, po.Organizations
		}
	}

	return false, []*contracts.Organization{}
}
