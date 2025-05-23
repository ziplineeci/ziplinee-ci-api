package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

// InjectStages injects some mandatory and configured stages
func InjectStages(config *APIConfig, mft manifest.ZiplineeManifest, builderTrack, gitSource, gitBranch string, supportsBuildStatus bool) (injectedManifest manifest.ZiplineeManifest, err error) {

	// get preferences for defaults
	var preferences *manifest.ZiplineeManifestPreferences
	if config != nil && config.ManifestPreferences != nil {
		preferences = config.ManifestPreferences
	} else {
		preferences = manifest.GetDefaultManifestPreferences()
	}

	// set preferences DefaultBranch to main if this happens to be used at this moment, so it gets set in the triggers correctly
	// todo figure out the default branch if a non-default branch is built
	if gitBranch == "main" {
		preferences.DefaultBranch = "main"
	}

	operatingSystem := getOperatingSystem(mft, *preferences)

	injectedManifest = mft

	// inject build stages
	injectedManifest.Stages = injectBuildStagesBefore(config, operatingSystem, injectedManifest, builderTrack, gitSource, supportsBuildStatus)
	injectedManifest.Stages = injectBuildStagesAfter(config, operatingSystem, injectedManifest, builderTrack, gitSource, supportsBuildStatus)

	// inject release stages
	for _, r := range injectedManifest.Releases {
		releaseOperatingSystem := operatingSystem
		if r.Builder != nil && r.Builder.OperatingSystem != "" {
			releaseOperatingSystem = r.Builder.OperatingSystem
		}

		r.Stages = injectReleaseStagesBefore(config, releaseOperatingSystem, injectedManifest, *r, builderTrack, gitSource, supportsBuildStatus)
		r.Stages = injectReleaseStagesAfter(config, releaseOperatingSystem, injectedManifest, *r, builderTrack, gitSource, supportsBuildStatus)
	}

	// inject bot stages
	for _, b := range injectedManifest.Bots {
		botOperatingSystem := operatingSystem
		if b.Builder != nil && b.Builder.OperatingSystem != "" {
			botOperatingSystem = b.Builder.OperatingSystem
		}

		b.Stages = injectBotStagesBefore(config, botOperatingSystem, injectedManifest, *b, builderTrack, gitSource, supportsBuildStatus)
		b.Stages = injectBotStagesAfter(config, botOperatingSystem, injectedManifest, *b, builderTrack, gitSource, supportsBuildStatus)
	}

	// ensure all injected stages have defaults for shell and working directory matching the target operating system
	injectedManifest.SetDefaults(*preferences)

	return
}

func getInjectedStageName(stageBaseName string, stages []*manifest.ZiplineeStage) string {

	injectedStageName := stageBaseName
	if stageExists(stages, injectedStageName) {
		i := 0
		for stageExists(stages, injectedStageName) {
			injectedStageName = fmt.Sprintf("%v-%v", stageBaseName, i)
			i++
		}
	}

	return injectedStageName
}

func injectIfNotExists(mft manifest.ZiplineeManifest, stages, parallelStages []*manifest.ZiplineeStage, stageToInject ...*manifest.ZiplineeStage) []*manifest.ZiplineeStage {

	for _, sti := range stageToInject {
		if !stageExists(stages, sti.Name) && labelSelectorMatches(mft, *sti) {
			parallelStages = append(parallelStages, sti)
		}
	}

	return parallelStages
}

func injectBuildStagesBefore(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = mft.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-before-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Build != nil && injectedStages.Build.Before != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Build.Before...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append([]*manifest.ZiplineeStage{softInjectedStage}, stages...)
		}
	}

	hardInjectedStage := &manifest.ZiplineeStage{
		Name:           getInjectedStageName("injected-before-hardcoded", stages),
		ParallelStages: []*manifest.ZiplineeStage{},
		AutoInjected:   true,
	}

	hardInjectedStage.ParallelStages = injectIfNotExists(mft, stages, hardInjectedStage.ParallelStages, &manifest.ZiplineeStage{
		Name:           "git-clone",
		ContainerImage: fmt.Sprintf("extensionci/git-clone:%v", builderTrack),
	})

	if supportsBuildStatus {
		hardInjectedStage.ParallelStages = injectIfNotExists(mft, stages, hardInjectedStage.ParallelStages, &manifest.ZiplineeStage{
			Name:           "set-pending-build-status",
			ContainerImage: fmt.Sprintf("extensionci/%v-status:%v", gitSource, builderTrack),
			CustomProperties: map[string]interface{}{
				"status": "pending",
			},
		})
	}

	if len(hardInjectedStage.ParallelStages) > 0 {
		for _, ps := range hardInjectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append([]*manifest.ZiplineeStage{hardInjectedStage}, stages...)
	}

	return stages
}

func injectBuildStagesAfter(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = mft.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {

		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-after-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
			When:           "status == 'succeeded' || status == 'failed'",
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Build != nil && injectedStages.Build.After != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Build.After...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append(stages, softInjectedStage)
		}
	}

	hardInjectedStage := &manifest.ZiplineeStage{
		Name:           getInjectedStageName("injected-after-hardcoded", stages),
		ParallelStages: []*manifest.ZiplineeStage{},
		AutoInjected:   true,
		When:           "status == 'succeeded' || status == 'failed'",
	}

	if supportsBuildStatus {
		hardInjectedStage.ParallelStages = injectIfNotExists(mft, stages, hardInjectedStage.ParallelStages, &manifest.ZiplineeStage{
			Name:           "set-build-status",
			ContainerImage: fmt.Sprintf("extensionci/%v-status:%v", gitSource, builderTrack),
			When:           "status == 'succeeded' || status == 'failed'",
		})
	}

	if len(hardInjectedStage.ParallelStages) > 0 {
		for _, ps := range hardInjectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append(stages, hardInjectedStage)
	}

	return stages
}

func injectReleaseStagesBefore(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, release manifest.ZiplineeRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = release.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-before-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Release != nil && injectedStages.Release.Before != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Release.Before...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append([]*manifest.ZiplineeStage{softInjectedStage}, stages...)
		}
	}

	hardInjectedStage := &manifest.ZiplineeStage{
		Name:           getInjectedStageName("injected-before-hardcoded", stages),
		ParallelStages: []*manifest.ZiplineeStage{},
		AutoInjected:   true,
	}

	if release.CloneRepository != nil && *release.CloneRepository {
		hardInjectedStage.ParallelStages = injectIfNotExists(mft, stages, hardInjectedStage.ParallelStages, &manifest.ZiplineeStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensionci/git-clone:%v", builderTrack),
		})
	}

	if len(hardInjectedStage.ParallelStages) > 0 {
		for _, ps := range hardInjectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append([]*manifest.ZiplineeStage{hardInjectedStage}, stages...)
	}

	return stages
}

func injectReleaseStagesAfter(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, release manifest.ZiplineeRelease, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = release.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-after-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
			When:           "status == 'succeeded' || status == 'failed'",
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Release != nil && injectedStages.Release.After != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Release.After...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append(stages, softInjectedStage)
		}
	}

	return stages
}

func injectBotStagesBefore(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, bot manifest.ZiplineeBot, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = bot.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-before-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Bot != nil && injectedStages.Bot.Before != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Bot.Before...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append([]*manifest.ZiplineeStage{softInjectedStage}, stages...)
		}
	}

	hardInjectedStage := &manifest.ZiplineeStage{
		Name:           getInjectedStageName("injected-before-hardcoded", stages),
		ParallelStages: []*manifest.ZiplineeStage{},
		AutoInjected:   true,
	}

	if bot.CloneRepository != nil && *bot.CloneRepository {
		hardInjectedStage.ParallelStages = injectIfNotExists(mft, stages, hardInjectedStage.ParallelStages, &manifest.ZiplineeStage{
			Name:           "git-clone",
			ContainerImage: fmt.Sprintf("extensionci/git-clone:%v", builderTrack),
		})
	}

	if len(hardInjectedStage.ParallelStages) > 0 {
		for _, ps := range hardInjectedStage.ParallelStages {
			ps.AutoInjected = true
		}
		stages = append([]*manifest.ZiplineeStage{hardInjectedStage}, stages...)
	}

	return stages
}

func injectBotStagesAfter(config *APIConfig, operatingSystem manifest.OperatingSystem, mft manifest.ZiplineeManifest, bot manifest.ZiplineeBot, builderTrack, gitSource string, supportsBuildStatus bool) (stages []*manifest.ZiplineeStage) {

	stages = bot.Stages

	// add any configured injected stages
	if config != nil && config.APIServer != nil && config.APIServer.InjectStagesPerOperatingSystem != nil {
		softInjectedStage := &manifest.ZiplineeStage{
			Name:           getInjectedStageName("injected-after-configured", stages),
			ParallelStages: []*manifest.ZiplineeStage{},
			AutoInjected:   true,
			When:           "status == 'succeeded' || status == 'failed'",
		}

		if injectedStages, found := config.APIServer.InjectStagesPerOperatingSystem[operatingSystem]; found && injectedStages.Bot != nil && injectedStages.Bot.After != nil {
			softInjectedStage.ParallelStages = injectIfNotExists(mft, stages, softInjectedStage.ParallelStages, injectedStages.Bot.After...)
		}

		if len(softInjectedStage.ParallelStages) > 0 {
			for _, ps := range softInjectedStage.ParallelStages {
				ps.AutoInjected = true
			}
			stages = append(stages, softInjectedStage)
		}
	}

	return stages
}

func stageExists(stages []*manifest.ZiplineeStage, stageName string) bool {
	for _, stage := range stages {
		if stage.Name == stageName {
			return true
		}
	}
	return false
}

func labelSelectorMatches(mft manifest.ZiplineeManifest, stage manifest.ZiplineeStage) bool {
	if val, ok := stage.CustomProperties["labelSelector"]; ok {
		labelSelector, ok := val.(map[string]interface{})
		if ok {
			allLabelsMatch := true
			for labelSelectorKey, labelSelectorValue := range labelSelector {

				// check if label exists in manifest
				if labelValue, labelExists := mft.Labels[labelSelectorKey]; labelExists {

					labelSelectorValueString := fmt.Sprintf("%v", labelSelectorValue)

					pattern := fmt.Sprintf("^%v$", strings.TrimSpace(labelSelectorValueString))

					log.Debug().Msgf("match %v: %v against %v", labelSelectorKey, labelValue, pattern)

					match, err := regexp.MatchString(pattern, labelValue)
					if err != nil {
						log.Fatal().Err(err).Msgf("Matching %v: %v against %v failed", labelSelectorKey, labelValue, pattern)
					}

					if !match {
						log.Debug().Msgf("%v: %v does not match %v", labelSelectorKey, labelValue, pattern)

						allLabelsMatch = false
						break
					}
				} else {
					return false
				}
			}

			return allLabelsMatch
		} else {
			log.Fatal().Msg("Can't cast labelSelector to map[string]string")
		}
	}

	return true
}

func getOperatingSystem(mft manifest.ZiplineeManifest, preferences manifest.ZiplineeManifestPreferences) manifest.OperatingSystem {

	if mft.Builder.OperatingSystem == "" {
		return preferences.BuilderOperatingSystems[0]
	}

	return mft.Builder.OperatingSystem
}

// InjectCommands injects configured commands
func InjectCommands(config *APIConfig, mft manifest.ZiplineeManifest) (injectedManifest manifest.ZiplineeManifest) {

	injectedManifest = mft

	if config == nil || config.APIServer == nil || config.APIServer.InjectCommandsPerOperatingSystemAndShell == nil {
		return
	}

	// get preferences for defaults
	var preferences *manifest.ZiplineeManifestPreferences
	if config != nil && config.ManifestPreferences != nil {
		preferences = config.ManifestPreferences
	} else {
		preferences = manifest.GetDefaultManifestPreferences()
	}

	operatingSystem := getOperatingSystem(injectedManifest, *preferences)

	// inject build stages
	injectedManifest.Stages = injectCommandsIntoStages(injectedManifest.Stages, config.APIServer.InjectCommandsPerOperatingSystemAndShell, operatingSystem)

	// inject release stages
	for _, r := range injectedManifest.Releases {
		releaseOperatingSystem := operatingSystem
		if r.Builder != nil && r.Builder.OperatingSystem != manifest.OperatingSystemUnknown {
			releaseOperatingSystem = r.Builder.OperatingSystem
		}

		r.Stages = injectCommandsIntoStages(r.Stages, config.APIServer.InjectCommandsPerOperatingSystemAndShell, releaseOperatingSystem)
	}

	return
}

func injectCommandsIntoStages(stages []*manifest.ZiplineeStage, commandsPerOperatingSystemAndShell map[manifest.OperatingSystem]map[string]InjectCommandsConfig, operatingSystem manifest.OperatingSystem) (injectedStages []*manifest.ZiplineeStage) {

	injectedStages = stages

	for _, s := range injectedStages {
		s = injectCommandsIntoStage(s, commandsPerOperatingSystemAndShell, operatingSystem)

		for _, ps := range s.ParallelStages {
			_ = injectCommandsIntoStage(ps, commandsPerOperatingSystemAndShell, operatingSystem)
		}
	}

	return
}

func injectCommandsIntoStage(stage *manifest.ZiplineeStage, commandsPerOperatingSystemAndShell map[manifest.OperatingSystem]map[string]InjectCommandsConfig, operatingSystem manifest.OperatingSystem) (injectedStage *manifest.ZiplineeStage) {

	injectedStage = stage

	if len(injectedStage.Commands) > 0 {
		// lookup if there's any before commands for os and shell
		if osConfig, ok := commandsPerOperatingSystemAndShell[operatingSystem]; ok {
			if shellConfig, ok := osConfig[injectedStage.Shell]; ok {
				if len(shellConfig.Before) > 0 {
					injectedStage.Commands = append(shellConfig.Before, injectedStage.Commands...)
				}
				if len(shellConfig.After) > 0 {
					injectedStage.Commands = append(injectedStage.Commands, shellConfig.After...)
				}
			}
		}
	}

	return
}
