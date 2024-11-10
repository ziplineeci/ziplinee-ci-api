package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	crypt "github.com/ziplineeci/ziplinee-ci-crypt"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

var (
	helper = NewWarningHelper(crypt.NewSecretHelper("SazbwMf3NZxVVbBqQHebPcXCqrVn3DDp", false))
)

func TestGetManifestWarnings(t *testing.T) {

	t.Run("ReturnsNoWarningsIfStageContainerImagesUsePinnedVersions", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "golang:1.12.4-alpine3.9",
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsNoWarningsIfStageContainerImageIsEmptyDueToNestedStages", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "",
					ParallelStages: []*manifest.ZiplineeStage{
						&manifest.ZiplineeStage{
							Name:           "build",
							ContainerImage: "golang:1.12.4-alpine3.9",
						},
					},
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesLatestTag", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "golang:latest",
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use **latest** or no tag for its container image: `build`; it is [best practice](https://ziplinee.io/usage/best-practices/#pin-image-versions) to pin stage images to specific versions so you don't spend hours tracking down build failures because the used image has changed.", warnings[0].Message)
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesNoTag", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "golang",
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use **latest** or no tag for its container image: `build`; it is [best practice](https://ziplinee.io/usage/best-practices/#pin-image-versions) to pin stage images to specific versions so you don't spend hours tracking down build failures because the used image has changed.", warnings[0].Message)
	})

	t.Run("ReturnsWarningIfStageContainerImageUsesExtensionWithDevTag", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "extensions/docker:dev",
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline has one or more stages that use the **dev** tag for its container image: `build`; it is [best practice](https://ziplinee.io/usage/best-practices/#avoid-using-ziplinee-s-dev-or-beta-tags) to avoid the dev tag alltogether, since it can be broken at any time.", warnings[0].Message)
	})

	t.Run("ReturnsNoWarningIfStageContainerImageUsesDevTagForNonExtensionImage", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "golang:dev",
				},
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsNoWarningIfStageContainerImageUsesExtensionWithDevTagForAnZiplineeBuild", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Stages: []*manifest.ZiplineeStage{
				&manifest.ZiplineeStage{
					Name:           "build",
					ContainerImage: "extensions/docker:dev",
				},
			},
		}
		fullRepoPath := "github.com/ziplineeci/ziplinee-ci-api"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsWarningIfBuilderTrackIsSetToDev", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			Builder: manifest.ZiplineeBuilder{
				Track: "dev",
			},
		}
		fullRepoPath := "github.com/kubernetes/kubernetes"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline uses the **dev** track for the builder; it is [best practice](https://ziplinee.io/usage/best-practices/#avoid-using-ziplinee-s-builder-dev-track) to avoid the dev track, since it can be broken at any time.", warnings[0].Message)
	})

	t.Run("ReturnsWarningIfManifestHasGlobalSecrets", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			GlobalEnvVars: map[string]string{
				"my-secret": "ziplinee.secret(deFTz5Bdjg6SUe29.oPIkXbze5G9PNEWS2-ZnArl8BCqHnx4MdTdxHg37th9u)",
			},
		}
		fullRepoPath := "github.com/ziplineeci/ziplinee-ci-api"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 1, len(warnings))
		assert.Equal(t, "warning", warnings[0].Status)
		assert.Equal(t, "This pipeline uses _global_ secrets which can be used by any pipeline; it is [best practice](https://ziplinee.io/usage/best-practices/#use-pipeline-restricted-secrets-instead-of-global-secrets) to use _restricted_ secrets instead, that can only be used by this pipeline. Please rotate the value stored in the secret and create a new one in the pipeline's secrets tab.", warnings[0].Message)
	})

	t.Run("ReturnsNoWarningIfManifestOnlyHasRestrictedSecrets", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			GlobalEnvVars: map[string]string{
				"my-secret": "ziplinee.secret(7pB-Znp16my5l-Gz.l--UakUaK5N8KYFt-sVNUaOY5uobSpWabJNVXYDEyDWT.hO6JcRARdtB-PY577NJeUrKMVOx-sjg617wTd8IkAh-PvIm9exuATeDeFiYaEr9eQtfreBQ=)",
			},
		}
		fullRepoPath := "github.com/ziplineeci/ziplinee-ci-api"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("ReturnsNoWarningIfManifestOnlyHasNoSecrets", func(t *testing.T) {

		mft := &manifest.ZiplineeManifest{
			GlobalEnvVars: map[string]string{
				"my-not-so-secret": "oops i forgot to encrypt",
			},
		}
		fullRepoPath := "github.com/ziplineeci/ziplinee-ci-api"

		// act
		warnings, err := helper.GetManifestWarnings(mft, fullRepoPath)

		assert.Nil(t, err)
		assert.Equal(t, 0, len(warnings))
	})
}

func TestGetContainerImageParts(t *testing.T) {

	t.Run("ReturnsEmptyRepoIfOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "", repo)
	})

	t.Run("ReturnsPathAsNameIfOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "golang", name)
	})

	t.Run("ReturnsTagIfProvidedForOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang:1.12.4-alpine3.9"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "1.12.4-alpine3.9", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForOfficialDockerHubImage", func(t *testing.T) {

		containerImage := "golang"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})

	t.Run("ReturnsPartUpToNameIfRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "ziplinee/ci-builder:dev"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "ziplinee", repo)
	})

	t.Run("ReturnsPathAsNameIfRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "ziplinee/ci-builder:dev"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "ci-builder", name)
	})

	t.Run("ReturnsTagIfProvidedForRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "ziplinee/ci-builder:dev"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "dev", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForRepositoryDockerHubImage", func(t *testing.T) {

		containerImage := "ziplinee/ci-builder"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})

	t.Run("ReturnsPartUpToNameIfGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/ziplinee/ci-builder:dev"

		// act
		repo, _, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "gcr.io/ziplinee", repo)
	})

	t.Run("ReturnsPathAsNameIfGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/ziplinee/ci-builder:dev"

		// act
		_, name, _ := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "ci-builder", name)
	})

	t.Run("ReturnsTagIfProvidedForGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/ziplinee/ci-builder:dev"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "dev", tag)
	})

	t.Run("ReturnsLatestTagIfNoTagProvidedForGoogleContainerRegistryImage", func(t *testing.T) {

		containerImage := "gcr.io/ziplinee/ci-builder"

		// act
		_, _, tag := helper.GetContainerImageParts(containerImage)

		assert.Equal(t, "latest", tag)
	})
}
