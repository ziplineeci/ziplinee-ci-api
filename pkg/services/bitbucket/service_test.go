package bitbucket

import (
	"context"
	"errors"
	"sync"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"

	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/bitbucketapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/pubsubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/queue"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
)

func TestCreateJobForBitbucketPush(t *testing.T) {

	t.Run("ReturnsErrNonCloneableEventIfPushEventHasNoChanges", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventChangeHasNoNewObject", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: nil,
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventNewTypeDoesNotEqualBranch", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "notbranch",
						},
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("ReturnsErrNonCloneableEventIfPushEventNewTargetHasNoHash", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "",
							},
						},
					},
				},
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.NotNil(t, err)
		assert.True(t, errors.Is(err, ErrNonCloneableEvent))
	})

	t.Run("CallsGetAccessTokenOnBitbucketAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "bitbucket.org/"})).AnyTimes()
		bitbucketapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
		}

		bitbucketapiClient.
			EXPECT().
			GetAccessTokenByInstallation(gomock.Any(), installation).
			Times(1)

		// act
		_ = service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)
	})

	t.Run("CallsGetZiplineeManifestOnBitbucketAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
			BuildControl: &api.BuildControl{},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		bitbucketapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			}).Times(1)

		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "bitbucket.org/ziplineeci/ziplinee-in-bitbucket"})).AnyTimes()
		bitbucketapiClient.EXPECT().GetAccessTokenByInstallation(gomock.Any(), gomock.Any()).AnyTimes()
		ziplineeService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetInstallationBySlug(gomock.Any(), gomock.Any()).Return(&bitbucketapi.BitbucketAppInstallation{}, nil)

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "ziplineeci/ziplinee-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.Nil(t, err)
	})

	t.Run("CallsCreateBuildOnZiplineeService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		bitbucketapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})

		ziplineeService.
			EXPECT().
			CreateBuild(gomock.Any(), gomock.Any()).
			Times(1)
		bitbucketapiClient.EXPECT().GetAccessTokenByInstallation(gomock.Any(), gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "bitbucket.org/ziplineeci/ziplinee-in-bitbucket"})).AnyTimes()
		bitbucketapiClient.EXPECT().GetInstallationBySlug(gomock.Any(), gomock.Any()).Return(&bitbucketapi.BitbucketAppInstallation{}, nil)

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "ziplineeci/ziplinee-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.Nil(t, err)
	})

	t.Run("PublishesGitTriggersOnTopic", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		bitbucketapiClient.EXPECT().GetAccessTokenByInstallation(gomock.Any(), gomock.Any()).AnyTimes()
		bitbucketapiClient.EXPECT().GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		pubsubapiClient.EXPECT().SubscribeToPubsubTriggers(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "bitbucket.org/"})).AnyTimes()

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
		}

		// act
		_ = service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)
	})

	t.Run("CallsSubscribeToPubsubTriggersOnPubsubAPIClient", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		bitbucketapiClient.
			EXPECT().
			GetZiplineeManifest(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, accesstoken bitbucketapi.AccessToken, event bitbucketapi.RepositoryPushEvent) (valid bool, manifest string, err error) {
				return true, "builder:\n  track: dev\n", nil
			})
		bitbucketapiClient.EXPECT().GetAccessTokenByInstallation(gomock.Any(), gomock.Any()).AnyTimes()
		ziplineeService.EXPECT().CreateBuild(gomock.Any(), gomock.Any()).AnyTimes()
		queueService.EXPECT().PublishGitEvent(gomock.Any(), gomock.Eq(manifest.ZiplineeGitEvent{Event: "push", Repository: "bitbucket.org/ziplineeci/ziplinee-in-bitbucket"})).AnyTimes()
		bitbucketapiClient.EXPECT().GetInstallationBySlug(gomock.Any(), gomock.Any()).Return(&bitbucketapi.BitbucketAppInstallation{}, nil)

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

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		installation := bitbucketapi.BitbucketAppInstallation{}
		pushEvent := bitbucketapi.RepositoryPushEvent{
			Push: bitbucketapi.PushEvent{
				Changes: []bitbucketapi.PushEventChange{
					{
						New: &bitbucketapi.PushEventChangeObject{
							Type: "branch",
							Target: bitbucketapi.PushEventChangeObjectTarget{
								Hash: "f0677f01cc6d54a5b042224a9eb374e98f979985",
							},
						},
					},
				},
			},
			Repository: bitbucketapi.Repository{
				FullName: "ziplineeci/ziplinee-in-bitbucket",
			},
		}

		// act
		err := service.CreateJobForBitbucketPush(context.Background(), installation, pushEvent)

		assert.Nil(t, err)
	})
}

func TestIsAllowedOwner(t *testing.T) {

	t.Run("ReturnsTrueIfInstallationIsKnown", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)
		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		bitbucketapiClient.EXPECT().GetInstallationBySlug(gomock.Any(), "anyone").Return(&bitbucketapi.BitbucketAppInstallation{}, nil)

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "anyone",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(context.Background(), &repository)

		assert.True(t, isAllowed)
	})

	t.Run("ReturnsFalseIfInstallationIsUnknown", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		bitbucketapiClient.EXPECT().GetInstallationBySlug(gomock.Any(), "ziplinee-in-bitbucket").Return(nil, bitbucketapi.ErrMissingInstallation)

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		repository := bitbucketapi.Repository{
			Owner: bitbucketapi.Owner{
				UserName: "ziplinee-in-bitbucket",
			},
		}

		// act
		isAllowed, _ := service.IsAllowedOwner(context.Background(), &repository)

		assert.False(t, isAllowed)
	})
}

func TestRename(t *testing.T) {

	t.Run("CallsRenameOnZiplineeService", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		config := &api.APIConfig{
			Integrations: &api.APIConfigIntegrations{
				Bitbucket: &api.BitbucketConfig{},
			},
		}
		bitbucketapiClient := bitbucketapi.NewMockClient(ctrl)
		pubsubapiClient := pubsubapi.NewMockClient(ctrl)
		ziplineeService := ziplinee.NewMockService(ctrl)
		queueService := queue.NewMockService(ctrl)

		ziplineeService.
			EXPECT().
			Rename(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(1)

		service := NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)

		// act
		err := service.Rename(context.Background(), "bitbucket.org", "ziplineeci", "ziplinee-ci-contracts", "bitbucket.org", "ziplineeci", "ziplinee-ci-protos")

		assert.Nil(t, err)
	})
}
