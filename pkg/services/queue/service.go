package queue

import (
	"context"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

//go:generate mockgen -package=queue -destination ./mock.go -source=service.go
type Service interface {
	CreateConnection(ctx context.Context) (err error)
	CloseConnection(ctx context.Context)
	InitSubscriptions(ctx context.Context) (err error)
	ReceiveCronEvent(cronEvent *manifest.ZiplineeCronEvent)
	ReceiveGitEvent(gitEvent *manifest.ZiplineeGitEvent)
	ReceiveGithubEvent(githubEvent *manifest.ZiplineeGithubEvent)
	ReceiveBitbucketEvent(bitbucketEvent *manifest.ZiplineeBitbucketEvent)
	PublishGitEvent(ctx context.Context, gitEvent manifest.ZiplineeGitEvent) (err error)
	PublishGithubEvent(ctx context.Context, githubEvent manifest.ZiplineeGithubEvent) (err error)
	PublishBitbucketEvent(ctx context.Context, bitbucketEvent manifest.ZiplineeBitbucketEvent) (err error)
}

// NewService returns a new ziplinee.Service
func NewService(config *api.APIConfig, ziplineeService ziplinee.Service) Service {
	return &service{
		config:          config,
		ziplineeService: ziplineeService,
	}
}

type service struct {
	config                *api.APIConfig
	ziplineeService       ziplinee.Service
	natsConnection        *nats.Conn
	natsEncodedConnection *nats.EncodedConn
}

func (s *service) CreateConnection(ctx context.Context) (err error) {
	s.natsConnection, err = nats.Connect(strings.Join(s.config.Queue.Hosts, ","))
	if err != nil {
		return
	}

	s.natsEncodedConnection, err = nats.NewEncodedConn(s.natsConnection, nats.JSON_ENCODER)
	if err != nil {
		return
	}

	return nil
}

func (s *service) CloseConnection(ctx context.Context) {
	if s.natsEncodedConnection != nil {
		s.natsEncodedConnection.Close()
	}
	if s.natsConnection != nil {
		s.natsConnection.Close()
	}
}

func (s *service) InitSubscriptions(ctx context.Context) (err error) {
	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectCron, "ziplinee-ci-api", s.ReceiveCronEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectGit, "ziplinee-ci-api", s.ReceiveGitEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectGithub, "ziplinee-ci-api", s.ReceiveGithubEvent)
	if err != nil {
		return
	}

	_, err = s.natsEncodedConnection.QueueSubscribe(s.config.Queue.SubjectBitbucket, "ziplinee-ci-api", s.ReceiveBitbucketEvent)
	if err != nil {
		return
	}

	return nil
}

func (s *service) ReceiveCronEvent(cronEvent *manifest.ZiplineeCronEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveCronEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.ziplineeService.FireCronTriggers(ctx, *cronEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling cron event from queue")
	}
}

func (s *service) ReceiveGitEvent(gitEvent *manifest.ZiplineeGitEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveGitEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	log.Debug().Msgf("Received git event: %v", gitEvent)
	err = s.ziplineeService.FireGitTriggers(ctx, *gitEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling git event from queue")
	}
}

func (s *service) ReceiveGithubEvent(githubEvent *manifest.ZiplineeGithubEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveGithubEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.ziplineeService.FireGithubTriggers(ctx, *githubEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling github event from queue")
	}
}

func (s *service) ReceiveBitbucketEvent(bitbucketEvent *manifest.ZiplineeBitbucketEvent) {
	var err error
	ctx := context.Background()
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "ReceiveBitbucketEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	err = s.ziplineeService.FireBitbucketTriggers(ctx, *bitbucketEvent)
	if err != nil {
		log.Error().Err(err).Msgf("Failed handling bitbucket event from queue")
	}
}

func (s *service) PublishGitEvent(ctx context.Context, gitEvent manifest.ZiplineeGitEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishGitEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectGit, &gitEvent)
}

func (s *service) PublishGithubEvent(ctx context.Context, githubEvent manifest.ZiplineeGithubEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishGithubEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectGithub, &githubEvent)
}

func (s *service) PublishBitbucketEvent(ctx context.Context, bitbucketEvent manifest.ZiplineeBitbucketEvent) (err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, api.GetSpanName("queue", "PublishBitbucketEvent"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.natsEncodedConnection.Publish(s.config.Queue.SubjectBitbucket, &bitbucketEvent)
}
