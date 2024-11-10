package cloudsource

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/cloudsourceapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/pubsubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/queue"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
	contracts "github.com/ziplineeci/ziplinee-ci-contracts"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

func TestCreateJobForCloudSourcePush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfNotificationHasNoRefUpdate", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "test-pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetAccessTokenOnCloudSourceAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetAccessToken(gomock.Any()).
			Times(1)

		cloudsourceapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/test-pubsub", Branch: "master"})).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "test-pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		_ = service.CreateJobForCloudSourcePush(context.Background(), notification)
	})

	t.Run("CallsGetZiplineeManifestOnCloudSourceAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			}).
			Times(1)

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		ziplineeService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/projects/test-project/repos/pubsub-test", Branch: "master"})).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})

	t.Run("CallsCreateBuildOnZiplineeService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		ziplineeService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, build contracts.Build) (b *contracts.Build, err error) {
				return
			}).
			Times(1)

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/projects/test-project/repos/pubsub-test", Branch: "master"})).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/projects/test-project/repos/pubsub-test", Branch: "master"})).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/projects/test-project/repos/pubsub-test", Branch: "master"})).AnyTimes()

		// act
		_ = service.CreateJobForCloudSourcePush(context.Background(), notification)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		cloudsourceapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken cloudsourceapi.AccessToken, event cloudsourceapi.PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		var wg sync.WaitGroup
		wg.Add(1)
		defer wg.Wait()
		pubsubapiClient.
			EXPECT().
			SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, manifestString string) (err error) {

				wg.Done()
				return
			}).
			Times(1)

		cloudsourceapiClient.EXPECT().GetAccessToken(gomock.Any()).AnyTimes()
		cloudsourceapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		ziplineeService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "source.developers.google.com/projects/test-project/repos/pubsub-test", Branch: "master"})).AnyTimes()

		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		err := service.CreateJobForCloudSourcePush(context.Background(), notification)

		assert.Nil(t, err)
	})
}

func TestIsAllowedProject(t *testing.T) {

	t.Run("ReturnsTrueIfAllowedProjectsConfigIsEmpty", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(context.Background(), notification)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfOwnerUsernameIsNotInAllowedProjectsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{
						{
							Project: "someone-else",
						},
					},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/test-project/repos/pubsub-test",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a025",
						NewId:      "f00768887da8de620612102",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(context.Background(), notification)

		assert.False(t, isAllowed)
	})

	t.Run("ReturnsTrueIfOwnerUsernameIsInAllowedProjectsConfig", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				CloudSource: &api.CloudSourceConfig{
					ProjectOrganizations: []api.ProjectOrganizations{
						{
							Project: "someone-else",
						},
						{
							Project: "ziplinee-in-cloudsource",
						},
					},
				},
			},
		}
		cloudsourceapiClient := cloudsourceapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, cloudsourceapiClient, pubsubapiClient, ziplineeService, queueService)

		notification := cloudsourceapi.PubSubNotification{
			Name:      "projects/ziplinee-in-cloudsource/repos/pubsub",
			Url:       "https://sourcecode.google.com",
			EventTime: time.Now(),
			RefUpdateEvent: &cloudsourceapi.RefUpdateEvent{
				Email: "ziplinee@ziplinee.com",
				RefUpdates: map[string]cloudsourceapi.RefUpdate{
					"refs/heads/master": cloudsourceapi.RefUpdate{
						RefName:    "refs/heads/master",
						UpdateType: "UPDATE_FAST_FORWARD",
						OldId:      "c7a28dd5de3403cc384a0258",
						NewId:      "f00768887da8de6206121029",
					},
				},
			},
		}

		// act
		isAllowed, _ := service.IsAllowedProject(context.Background(), notification)

		assert.True(t, isAllowed)
	})
}
