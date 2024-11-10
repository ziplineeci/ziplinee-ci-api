package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ziplineeci/ziplinee-ci-api/pkg/migrationpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	stdbigquery "cloud.google.com/go/bigquery"
	stdpubsub "cloud.google.com/go/pubsub"
	stdstorage "cloud.google.com/go/storage"
	"github.com/alecthomas/kingpin"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/bigquery"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/bitbucketapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/builderapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/cloudsourceapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/cloudstorage"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/database"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/dockerhubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/githubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/prometheus"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/pubsubapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/clients/slackapi"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/bitbucket"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/catalog"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/cloudsource"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/github"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/pubsub"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/queue"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/rbac"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/slack"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/services/ziplinee"
	crypt "github.com/ziplineeci/ziplinee-ci-crypt"
	foundation "github.com/ziplineeci/ziplinee-foundation"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sourcerepo/v1"
	stdsourcerepo "google.golang.org/api/sourcerepo/v1"
)

var (
	appgroup  string
	app       string
	version   string
	branch    string
	revision  string
	buildDate string
)

var (
	// flags
	apiAddress                   = kingpin.Flag("api-listen-address", "The address to listen on for api HTTP requests.").Default(":5000").String()
	configFilesPath              = kingpin.Flag("config-files-path", "The path to yaml config file configuring this application.").Default("/configs").Envar("CONFIG_FILES_PATH").String()
	templatesPath                = kingpin.Flag("templates-path", "The path to the manifest templates being used by the 'Create' functionality.").Default("/templates").Envar("TEMPLATES_PATH").String()
	secretDecryptionKeyPath      = kingpin.Flag("secret-decryption-key-path", "The path to the AES-256 key used to decrypt secrets that have been encrypted with it.").Default("/secrets/secretDecryptionKey").Envar("SECRET_DECRYPTION_KEY_PATH").String()
	jwtKeyPath                   = kingpin.Flag("jwt-key-path", "The path to 256 bit jwt key used for api authentication.").Default("/secrets/jwtKey").Envar("JWT_KEY_PATH").String()
	gracefulShutdownDelaySeconds = kingpin.Flag("graceful-shutdown-delay-seconds", "The number of seconds to wait with graceful shutdown in order to let endpoints update propagation finish.").Default("15").Envar("GRACEFUL_SHUTDOWN_DELAY_SECONDS").Int()
	gcsMigratorServer            = kingpin.Flag("gcs-migrator-server", "Address in the format of host:port should be same as in gcs-migrator/server.py").Default("localhost:50051").Envar("GCS_MIGRATOR_SERVER").String()
)

var (
	gcsMigratorRestartDelay = 1 * time.Minute
	gcsMigrator             *exec.Cmd
	gcsMigratorStartedAt    time.Time
)

func main() {

	// parse command line parameters
	kingpin.Parse()

	// init log format from envvar ZIPLINEE_LOG_FORMAT
	foundation.InitLoggingFromEnv(foundation.NewApplicationInfo(appgroup, app, version, branch, revision, buildDate))

	err := os.Mkdir("/tmp", 0744)
	if err != nil && !os.IsExist(err) {
		log.Fatal().Err(err).Msg("failed creating /tmp directory")
	}
	startGcsMigrator()
	closer := initJaeger()
	defer closer.Close()

	sigs, wg := foundation.InitGracefulShutdownHandling()
	stop := make(chan struct{}) // channel to signal goroutines to stop

	ctx := foundation.InitCancellationContext(context.Background())

	// start prometheus
	foundation.InitMetricsWithPort(9001)

	// handle api requests
	srv, queueService := initRequestHandlers(ctx, stop, wg)
	defer queueService.CloseConnection(ctx)

	log.Debug().Msg("Handling requests...")

	foundation.HandleGracefulShutdown(sigs, wg, func() {

		time.Sleep(time.Duration(*gracefulShutdownDelaySeconds) * time.Second)

		// shut down gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("Graceful server shutdown failed")
		}

		log.Debug().Msg("Stopping goroutines...")
		close(stop) // tell goroutines to stop themselves
	})
}

func initRequestHandlers(ctx context.Context, stopChannel <-chan struct{}, waitGroup *sync.WaitGroup) (*http.Server, queue.Service) {

	config, encryptedConfig, secretHelper := getConfig(ctx)
	bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService := getGoogleCloudClients(ctx, config)
	bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, databaseClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient := getClients(ctx, config, encryptedConfig, secretHelper, bqClient, pubsubClient, gcsClient, sourcerepoTokenSource, sourcerepoService)
	ziplineeService, queueService, rbacService, githubService, bitbucketService, cloudsourceService, catalogService := getServices(ctx, config, encryptedConfig, secretHelper, bigqueryClient, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, databaseClient, dockerhubapiClient, builderapiClient, cloudstorageClient, prometheusClient, cloudsourceClient)
	gcsMigratorHealthClient, gcsMigratorClient := getGcsMigratorClients()
	bitbucketHandler, githubHandler, ziplineeHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler := getHandlers(ctx, config, encryptedConfig, secretHelper, bitbucketapiClient, githubapiClient, slackapiClient, pubsubapiClient, databaseClient, builderapiClient, cloudstorageClient, ziplineeService, rbacService, githubService, bitbucketService, cloudsourceService, catalogService, gcsMigratorClient)

	waitGroup.Add(1)
	//go ziplineeHandler.PollMigrationTasks(stopChannel, waitGroup.Done)
	err := queueService.CreateConnection(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating connection to queue")
	}
	err = queueService.InitSubscriptions(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed initializing queue subscriptions")
	}

	srv := configureGinGonic(config, bitbucketHandler, githubHandler, ziplineeHandler, rbacHandler, pubsubHandler, slackHandler, cloudsourceHandler, catalogHandler, gcsMigratorHealthClient)

	// watch for config changes
	foundation.WatchForFileChanges(*configFilesPath, func(event fsnotify.Event) {
		log.Debug().Msgf("Configs at %v were updated, refreshing instances...", *configFilesPath)

		// refresh config
		newConfig, newEncryptedConfig, _ := getConfig(ctx)

		*config = *newConfig
		*encryptedConfig = *newEncryptedConfig

		// refresh google cloud clients
		newBqClient, newPubsubClient, newGcsClient, newSourcerepoTokenSource, newSourcerepoService := getGoogleCloudClients(ctx, config)

		if newBqClient != nil {
			*bqClient = *newBqClient
		}
		if newPubsubClient != nil {
			*pubsubClient = *newPubsubClient
		}
		if newGcsClient != nil {
			*gcsClient = *newGcsClient
		}
		if newPubsubClient != nil {
			*pubsubClient = *newPubsubClient
		}
		if newSourcerepoService != nil {
			*sourcerepoService = *newSourcerepoService
			sourcerepoTokenSource = newSourcerepoTokenSource
		}

	})

	// watch for service account key file changes
	foundation.WatchForFileChanges(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), func(event fsnotify.Event) {
		log.Info().Msg("Service account key file was updated, refreshing instances...")
		startGcsMigrator()

		// refresh google cloud clients
		newBqClient, newPubsubClient, newGcsClient, newSourcerepoTokenSource, newSourcerepoService := getGoogleCloudClients(ctx, config)

		*bqClient = *newBqClient
		*pubsubClient = *newPubsubClient
		*gcsClient = *newGcsClient
		sourcerepoTokenSource = newSourcerepoTokenSource
		*sourcerepoService = *newSourcerepoService
	})

	// watch for database client certificate changes
	if config != nil && config.Database != nil && !config.Database.Insecure && config.Database.CertificatePath != "" {
		foundation.WatchForFileChanges(config.Database.CertificatePath, func(event fsnotify.Event) {
			log.Debug().Msg("Database client certificate file was updated, refreshing connection...")

			err = databaseClient.Connect(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed connecting to database")
			}
		})
	}

	return srv, queueService
}

// !! Migration changes !!
func getGcsMigratorClients() (migrationpb.HealthClient, migrationpb.ServiceClient) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(*gcsMigratorServer, opts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to dial grpc connection with gcs-migrator: %v", err)
	}
	return migrationpb.NewHealthClient(conn), migrationpb.NewServiceClient(conn)
}

func getConfig(ctx context.Context) (*api.APIConfig, *api.APIConfig, crypt.SecretHelper) {

	// read decryption key from secretDecryptionKeyPath
	if !foundation.FileExists(*secretDecryptionKeyPath) {
		log.Fatal().Msgf("Cannot find secret decryption key at path %v", *secretDecryptionKeyPath)
	}

	// read jwt key from jwtKeyPath
	if !foundation.FileExists(*jwtKeyPath) {
		log.Fatal().Msgf("Cannot find jwt key at path %v", *jwtKeyPath)
	}

	secretDecryptionKeyBytes, err := os.ReadFile(*secretDecryptionKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed reading secret decryption key from path %v", *secretDecryptionKeyPath)
	}

	jwtKeyPathBytes, err := os.ReadFile(*jwtKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed reading jwt key from path %v", *jwtKeyPath)
	}

	log.Debug().Msg("Creating helpers...")

	secretHelper := crypt.NewSecretHelper(string(secretDecryptionKeyBytes), false)

	log.Debug().Msg("Creating config reader...")

	configReader := api.NewConfigReader(secretHelper, string(jwtKeyPathBytes))

	// await for config file to be present, due to git-sync sidecar startup it can take some time
	for {
		if foundation.DirExists(*configFilesPath) {
			configFilePaths, err := configReader.GetConfigFilePaths(*configFilesPath)
			if err != nil {
				log.Fatal().Err(err).Msgf("Failed getting config file paths inside directory %v", *configFilesPath)
			}
			if len(configFilePaths) > 0 {
				break
			}
			log.Warn().Msgf("No config files exist yet in directory %v", *configFilesPath)
		} else {
			log.Warn().Msgf("Config directory %v does not exist yet", *configFilesPath)
		}

		log.Debug().Msg("Sleeping for 5 seconds while config file is synced...")
		time.Sleep(5 * time.Second)
	}

	config, err := configReader.ReadConfigFromFiles(*configFilesPath, true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration")
	}

	encryptedConfig, err := configReader.ReadConfigFromFiles(*configFilesPath, false)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed reading configuration without decrypting")
	}

	return config, encryptedConfig, secretHelper
}

func getGoogleCloudClients(ctx context.Context, config *api.APIConfig) (bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, tokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) {

	log.Debug().Msg("Creating Google Cloud clients...")

	var err error

	if config != nil && config.Integrations != nil && config.Integrations.BigQuery != nil && config.Integrations.BigQuery.Enable {
		bqClient, err = stdbigquery.NewClient(ctx, config.Integrations.BigQuery.ProjectID)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating new BigQueryClient has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.Pubsub != nil && config.Integrations.Pubsub.Enable {
		pubsubClient, err = stdpubsub.NewClient(ctx, config.Integrations.Pubsub.DefaultProject)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google pubsub client has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.CloudStorage != nil && config.Integrations.CloudStorage.Enable {
		gcsClient, err = stdstorage.NewClient(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud storage client has failed")
		}
	}

	if config != nil && config.Integrations != nil && config.Integrations.CloudSource != nil && config.Integrations.CloudSource.Enable {
		tokenSource, err = google.DefaultTokenSource(ctx, sourcerepo.CloudPlatformScope)
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud token source has failed")
		}
		sourcerepoService, err = stdsourcerepo.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, tokenSource)))
		if err != nil {
			log.Fatal().Err(err).Msg("Creating google cloud source repo service has failed")
		}
	}

	return bqClient, pubsubClient, gcsClient, tokenSource, sourcerepoService
}

func getClients(ctx context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bqClient *stdbigquery.Client, pubsubClient *stdpubsub.Client, gcsClient *stdstorage.Client, sourcerepoTokenSource oauth2.TokenSource, sourcerepoService *stdsourcerepo.Service) (bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, databaseClient database.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) {

	log.Debug().Msg("Creating clients...")

	// creates the in-cluster config
	kubeClientConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed getting in-cluster kubernetes config")
	}
	// creates the clientset
	kubeClientset, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating kubernetes clientset")
	}

	// bigquery client
	bigqueryClient = bigquery.NewClient(config, bqClient)
	bigqueryClient = bigquery.NewTracingClient(bigqueryClient)
	bigqueryClient = bigquery.NewLoggingClient(bigqueryClient)
	bigqueryClient = bigquery.NewMetricsClient(bigqueryClient,
		api.NewRequestCounter("bigquery_client"),
		api.NewRequestHistogram("bigquery_client"),
	)
	err = bigqueryClient.Init(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Initializing BigQuery tables has failed")
	}

	// bitbucketapi client
	bitbucketapiClient = bitbucketapi.NewClient(config, kubeClientset, secretHelper)
	bitbucketapiClient = bitbucketapi.NewTracingClient(bitbucketapiClient)
	bitbucketapiClient = bitbucketapi.NewLoggingClient(bitbucketapiClient)
	bitbucketapiClient = bitbucketapi.NewMetricsClient(bitbucketapiClient,
		api.NewRequestCounter("bitbucketapi_client"),
		api.NewRequestHistogram("bitbucketapi_client"),
	)

	// githubapi client
	githubapiClient = githubapi.NewClient(config, kubeClientset, secretHelper)
	githubapiClient = githubapi.NewTracingClient(githubapiClient)
	githubapiClient = githubapi.NewLoggingClient(githubapiClient)
	githubapiClient = githubapi.NewMetricsClient(githubapiClient,
		api.NewRequestCounter("githubapi_client"),
		api.NewRequestHistogram("githubapi_client"),
	)

	// slackapi client
	slackapiClient = slackapi.NewClient(config)
	slackapiClient = slackapi.NewTracingClient(slackapiClient)
	slackapiClient = slackapi.NewLoggingClient(slackapiClient)
	slackapiClient = slackapi.NewMetricsClient(slackapiClient,
		api.NewRequestCounter("slackapi_client"),
		api.NewRequestHistogram("slackapi_client"),
	)

	// pubsubapi client
	pubsubapiClient = pubsubapi.NewClient(config, pubsubClient)
	pubsubapiClient = pubsubapi.NewTracingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewLoggingClient(pubsubapiClient)
	pubsubapiClient = pubsubapi.NewMetricsClient(pubsubapiClient,
		api.NewRequestCounter("pubsubapi_client"),
		api.NewRequestHistogram("pubsubapi_client"),
	)

	// cockroachdb client
	databaseClient = database.NewClient(config)
	databaseClient = database.NewTracingClient(databaseClient)
	databaseClient = database.NewLoggingClient(databaseClient)
	databaseClient = database.NewMetricsClient(databaseClient,
		api.NewRequestCounter("cockroachdb_client"),
		api.NewRequestHistogram("cockroachdb_client"),
	)
	err = databaseClient.Connect(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed connecting to database")
	}

	err = databaseClient.AwaitDatabaseReadiness(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed waiting for database to be ready")
	}

	// dockerhubapi client
	dockerhubapiClient = dockerhubapi.NewClient()
	dockerhubapiClient = dockerhubapi.NewTracingClient(dockerhubapiClient)
	dockerhubapiClient = dockerhubapi.NewLoggingClient(dockerhubapiClient)
	dockerhubapiClient = dockerhubapi.NewMetricsClient(dockerhubapiClient,
		api.NewRequestCounter("dockerhubapi_client"),
		api.NewRequestHistogram("dockerhubapi_client"),
	)

	// builderapi client
	builderapiClient = builderapi.NewClient(config, encryptedConfig, secretHelper, kubeClientset, dockerhubapiClient)
	builderapiClient = builderapi.NewTracingClient(builderapiClient)
	builderapiClient = builderapi.NewLoggingClient(builderapiClient)
	builderapiClient = builderapi.NewMetricsClient(builderapiClient,
		api.NewRequestCounter("builderapi_client"),
		api.NewRequestHistogram("builderapi_client"),
	)

	// cloudstorage client
	cloudstorageClient = cloudstorage.NewClient(config, gcsClient)
	cloudstorageClient = cloudstorage.NewTracingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewLoggingClient(cloudstorageClient)
	cloudstorageClient = cloudstorage.NewMetricsClient(cloudstorageClient,
		api.NewRequestCounter("cloudstorage_client"),
		api.NewRequestHistogram("cloudstorage_client"),
	)

	// prometheus client
	prometheusClient = prometheus.NewClient(config)
	prometheusClient = prometheus.NewTracingClient(prometheusClient)
	prometheusClient = prometheus.NewLoggingClient(prometheusClient)
	prometheusClient = prometheus.NewMetricsClient(prometheusClient,
		api.NewRequestCounter("prometheus_client"),
		api.NewRequestHistogram("prometheus_client"),
	)

	// cloudsourceapi client
	cloudsourceClient = cloudsourceapi.NewClient(config, sourcerepoTokenSource, sourcerepoService)
	cloudsourceClient = cloudsourceapi.NewTracingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewLoggingClient(cloudsourceClient)
	cloudsourceClient = cloudsourceapi.NewMetricsClient(cloudsourceClient,
		api.NewRequestCounter("cloudsource_client"),
		api.NewRequestHistogram("cloudsource_client"),
	)

	return
}

func getServices(ctx context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bigqueryClient bigquery.Client, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, databaseClient database.Client, dockerhubapiClient dockerhubapi.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, prometheusClient prometheus.Client, cloudsourceClient cloudsourceapi.Client) (ziplineeService ziplinee.Service, queueService queue.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service, catalogService catalog.Service) {

	log.Debug().Msg("Creating services...")

	// ziplinee service
	ziplineeService = ziplinee.NewService(config, databaseClient, secretHelper, prometheusClient, cloudstorageClient, builderapiClient, githubapiClient.JobVarsFunc(ctx), bitbucketapiClient.JobVarsFunc(ctx), cloudsourceClient.JobVarsFunc(ctx))
	ziplineeService = ziplinee.NewTracingService(ziplineeService)
	ziplineeService = ziplinee.NewLoggingService(ziplineeService)
	ziplineeService = ziplinee.NewMetricsService(ziplineeService,
		api.NewRequestCounter("ziplinee_service"),
		api.NewRequestHistogram("ziplinee_service"),
	)

	// queue service
	queueService = queue.NewService(config, ziplineeService)

	// rbac service
	rbacService = rbac.NewService(config, databaseClient)
	rbacService = rbac.NewTracingService(rbacService)
	rbacService = rbac.NewLoggingService(rbacService)
	rbacService = rbac.NewMetricsService(rbacService,
		api.NewRequestCounter("rbac_service"),
		api.NewRequestHistogram("rbac_service"),
	)

	// github service
	githubService = github.NewService(config, githubapiClient, pubsubapiClient, ziplineeService, queueService)
	githubService = github.NewTracingService(githubService)
	githubService = github.NewLoggingService(githubService)
	githubService = github.NewMetricsService(githubService,
		api.NewRequestCounter("github_service"),
		api.NewRequestHistogram("github_service"),
	)

	// bitbucket service
	bitbucketService = bitbucket.NewService(config, bitbucketapiClient, pubsubapiClient, ziplineeService, queueService)
	bitbucketService = bitbucket.NewTracingService(bitbucketService)
	bitbucketService = bitbucket.NewLoggingService(bitbucketService)
	bitbucketService = bitbucket.NewMetricsService(bitbucketService,
		api.NewRequestCounter("bitbucket_service"),
		api.NewRequestHistogram("bitbucket_service"),
	)

	// cloudsource service
	cloudsourceService = cloudsource.NewService(config, cloudsourceClient, pubsubapiClient, ziplineeService, queueService)
	cloudsourceService = cloudsource.NewTracingService(cloudsourceService)
	cloudsourceService = cloudsource.NewLoggingService(cloudsourceService)
	cloudsourceService = cloudsource.NewMetricsService(cloudsourceService,
		api.NewRequestCounter("cloudsource_service"),
		api.NewRequestHistogram("cloudsource_service"),
	)

	// catalog service
	catalogService = catalog.NewService(config, databaseClient)
	catalogService = catalog.NewTracingService(catalogService)
	catalogService = catalog.NewLoggingService(catalogService)
	catalogService = catalog.NewMetricsService(catalogService,
		api.NewRequestCounter("catalog_service"),
		api.NewRequestHistogram("catalog_service"),
	)

	return
}

func getHandlers(_ context.Context, config *api.APIConfig, encryptedConfig *api.APIConfig, secretHelper crypt.SecretHelper, bitbucketapiClient bitbucketapi.Client, githubapiClient githubapi.Client, slackapiClient slackapi.Client, pubsubapiClient pubsubapi.Client, databaseClient database.Client, builderapiClient builderapi.Client, cloudstorageClient cloudstorage.Client, ziplineeService ziplinee.Service, rbacService rbac.Service, githubService github.Service, bitbucketService bitbucket.Service, cloudsourceService cloudsource.Service, catalogService catalog.Service, gcsMigratorClient migrationpb.ServiceClient) (
	bitbucketHandler bitbucket.Handler, githubHandler github.Handler, ziplineeHandler ziplinee.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler, catalogHandler catalog.Handler) {
	log.Debug().Msg("Creating http handlers...")

	warningHelper := api.NewWarningHelper(secretHelper)

	// transport
	bitbucketHandler = bitbucket.NewHandler(bitbucketService, config, bitbucketapiClient)
	githubHandler = github.NewHandler(githubService, config, githubapiClient, databaseClient)
	ziplineeHandler = ziplinee.NewHandler(*templatesPath, config, encryptedConfig, databaseClient, cloudstorageClient, builderapiClient, ziplineeService, warningHelper, secretHelper, gcsMigratorClient)
	rbacHandler = rbac.NewHandler(config, rbacService, databaseClient, bitbucketapiClient, githubapiClient)
	pubsubHandler = pubsub.NewHandler(pubsubapiClient, ziplineeService)
	slackHandler = slack.NewHandler(secretHelper, config, slackapiClient, databaseClient, ziplineeService)
	cloudsourceHandler = cloudsource.NewHandler(pubsubapiClient, cloudsourceService)
	catalogHandler = catalog.NewHandler(config, catalogService, databaseClient)

	return
}

func configureGinGonic(config *api.APIConfig, bitbucketHandler bitbucket.Handler, githubHandler github.Handler, ziplineeHandler ziplinee.Handler, rbacHandler rbac.Handler, pubsubHandler pubsub.Handler, slackHandler slack.Handler, cloudsourceHandler cloudsource.Handler, catalogHandler catalog.Handler, gcsMigratorHealthClient migrationpb.HealthClient) *http.Server {

	// run gin in release mode and other defaults
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = log.Logger
	gin.DisableConsoleColor()

	// creates a router without any middleware
	log.Debug().Msg("Creating gin router...")
	router := gin.New()

	// recovery middleware recovers from any panics and writes a 500 if there was one.
	log.Debug().Msg("Adding recovery middleware...")
	// log panic as error so it gets shipped
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Error().Err(fmt.Errorf(err)).Msg("Recovered from panic")
		} else {
			log.Error().Interface("recovered", recovered).Msg("Recovered from panic without error")
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// opentracing middleware
	log.Debug().Msg("Adding opentracing middleware...")
	router.Use(api.OpenTracingMiddleware())

	// middleware to handle auth for different endpoints
	log.Debug().Msg("Adding auth middleware...")
	authMiddleware := api.NewAuthMiddleware(config)
	jwtMiddleware, err := authMiddleware.GinJWTMiddleware(rbacHandler.HandleOAuthLoginProviderAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware")
	}
	clientLoginJWTMiddleware, err := authMiddleware.GinJWTMiddlewareForClientLogin(rbacHandler.HandleClientLoginProviderAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware for client login")
	}
	impersonateJWTMiddleware, err := authMiddleware.GinJWTMiddleware(rbacHandler.HandleImpersonateAuthenticator())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed creating JWT middleware for user impersonation")
	}

	preZippedJWTMiddlewareRoutes := router.Group("/", jwtMiddleware.MiddlewareFunc())

	// Gzip and logging middleware
	log.Debug().Msg("Adding gzip middleware...")
	routes := router.Group("/", gzip.Gzip(gzip.BestSpeed))

	log.Debug().Msg("Setting up routes...")
	routes.POST("/api/integrations/github/events", githubHandler.Handle)
	routes.GET("/api/integrations/github/status", func(c *gin.Context) { c.String(200, "Github, I'm cool!") })
	routes.GET("/api/integrations/github/redirect", githubHandler.Redirect)

	routes.POST("/api/integrations/bitbucket/events", bitbucketHandler.Handle)
	routes.GET("/api/integrations/bitbucket/status", func(c *gin.Context) { c.String(200, "Bitbucket, I'm cool!") })
	routes.GET("/api/integrations/bitbucket/descriptor", bitbucketHandler.Descriptor)
	routes.POST("/api/integrations/bitbucket/installed", bitbucketHandler.Installed)
	routes.POST("/api/integrations/bitbucket/uninstalled", bitbucketHandler.Uninstalled)
	routes.GET("/api/integrations/bitbucket/redirect", bitbucketHandler.Redirect)

	routes.POST("/api/integrations/slack/slash", slackHandler.Handle)
	routes.GET("/api/integrations/slack/status", func(c *gin.Context) { c.String(200, "Slack, I'm cool!") })

	// google jwt auth protected endpoints
	googleAuthorizedRoutes := routes.Group("/", authMiddleware.GoogleJWTMiddlewareFunc())
	{
		googleAuthorizedRoutes.POST("/api/integrations/pubsub/events", pubsubHandler.PostPubsubEvent)
		googleAuthorizedRoutes.POST("/api/integrations/cloudsource/events", cloudsourceHandler.PostPubsubEvent)
	}
	routes.GET("/api/integrations/pubsub/status", func(c *gin.Context) { c.String(200, "Pub/Sub, I'm cool!") })
	routes.GET("/api/integrations/cloudsource/status", func(c *gin.Context) { c.String(200, "Cloud Source, I'm cool!") })

	// public routes for logging in
	routes.GET("/api/auth/providers", rbacHandler.GetProviders)
	routes.GET("/api/auth/login/:provider", rbacHandler.LoginProvider)
	routes.GET("/api/auth/logout", jwtMiddleware.LogoutHandler)
	routes.GET("/api/auth/handle/:provider", jwtMiddleware.LoginHandler)
	routes.POST("/api/auth/client/login", clientLoginJWTMiddleware.LoginHandler)
	routes.POST("/api/auth/client/logout", clientLoginJWTMiddleware.LogoutHandler)

	// routes that require to be logged in and have a valid jwt
	jwtMiddlewareRoutes := routes.Group("/", jwtMiddleware.MiddlewareFunc())
	{
		// logged in user endpoints
		jwtMiddlewareRoutes.GET("/api/me", rbacHandler.GetLoggedInUser)

		// actions
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds", ziplineeHandler.CreatePipelineBuild)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases", ziplineeHandler.CreatePipelineRelease)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/bots", ziplineeHandler.CreatePipelineBot)
		jwtMiddlewareRoutes.POST("/api/notifications", ziplineeHandler.CreateNotification)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", ziplineeHandler.CancelPipelineBuild)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/releases/:id", ziplineeHandler.CancelPipelineRelease)
		jwtMiddlewareRoutes.DELETE("/api/pipelines/:source/:owner/:repo/bots/:id", ziplineeHandler.CancelPipelineBot)

		// to be removed after changing web frontend to use the /api/admin routes
		jwtMiddlewareRoutes.GET("/api/roles", rbacHandler.GetRoles)

		jwtMiddlewareRoutes.GET("/api/users", rbacHandler.GetUsers)
		jwtMiddlewareRoutes.GET("/api/users/:id", rbacHandler.GetUser)
		jwtMiddlewareRoutes.POST("/api/users", rbacHandler.CreateUser)
		jwtMiddlewareRoutes.PUT("/api/users/:id", rbacHandler.UpdateUser)
		jwtMiddlewareRoutes.DELETE("/api/users/:id", rbacHandler.DeleteUser)

		jwtMiddlewareRoutes.GET("/api/groups", rbacHandler.GetGroups)
		jwtMiddlewareRoutes.GET("/api/groups/:id", rbacHandler.GetGroup)
		jwtMiddlewareRoutes.POST("/api/groups", rbacHandler.CreateGroup)
		jwtMiddlewareRoutes.PUT("/api/groups/:id", rbacHandler.UpdateGroup)
		jwtMiddlewareRoutes.DELETE("/api/groups/:id", rbacHandler.DeleteGroup)

		jwtMiddlewareRoutes.GET("/api/organizations", rbacHandler.GetOrganizations)
		jwtMiddlewareRoutes.GET("/api/organizations/:id", rbacHandler.GetOrganization)
		jwtMiddlewareRoutes.POST("/api/organizations", rbacHandler.CreateOrganization)
		jwtMiddlewareRoutes.PUT("/api/organizations/:id", rbacHandler.UpdateOrganization)
		jwtMiddlewareRoutes.DELETE("/api/organizations/:id", rbacHandler.DeleteOrganization)

		jwtMiddlewareRoutes.GET("/api/clients", rbacHandler.GetClients)
		jwtMiddlewareRoutes.GET("/api/clients/:id", rbacHandler.GetClient)
		jwtMiddlewareRoutes.POST("/api/clients", rbacHandler.CreateClient)
		jwtMiddlewareRoutes.PUT("/api/clients/:id", rbacHandler.UpdateClient)
		jwtMiddlewareRoutes.DELETE("/api/clients/:id", rbacHandler.DeleteClient)

		// admin section
		jwtMiddlewareRoutes.GET("/api/auth/impersonate/:id", impersonateJWTMiddleware.LoginHandler)
		jwtMiddlewareRoutes.GET("/api/admin/roles", rbacHandler.GetRoles)

		jwtMiddlewareRoutes.GET("/api/admin/users", rbacHandler.GetUsers)
		jwtMiddlewareRoutes.GET("/api/admin/users/:id", rbacHandler.GetUser)
		jwtMiddlewareRoutes.POST("/api/admin/users", rbacHandler.CreateUser)
		jwtMiddlewareRoutes.PUT("/api/admin/users/:id", rbacHandler.UpdateUser)
		jwtMiddlewareRoutes.DELETE("/api/admin/users/:id", rbacHandler.DeleteUser)

		jwtMiddlewareRoutes.GET("/api/admin/pipelines", rbacHandler.GetPipelines)
		jwtMiddlewareRoutes.GET("/api/admin/pipelines/:source/:owner/:repo", rbacHandler.GetPipeline)
		jwtMiddlewareRoutes.PUT("/api/admin/pipelines/:source/:owner/:repo", rbacHandler.UpdatePipeline)

		jwtMiddlewareRoutes.POST("/api/admin/batch/users", rbacHandler.BatchUpdateUsers)
		jwtMiddlewareRoutes.POST("/api/admin/batch/groups", rbacHandler.BatchUpdateGroups)
		jwtMiddlewareRoutes.POST("/api/admin/batch/organizations", rbacHandler.BatchUpdateOrganizations)
		jwtMiddlewareRoutes.POST("/api/admin/batch/clients", rbacHandler.BatchUpdateClients)
		jwtMiddlewareRoutes.POST("/api/admin/batch/pipelines", rbacHandler.BatchUpdatePipelines)

		jwtMiddlewareRoutes.GET("/api/admin/groups", rbacHandler.GetGroups)
		jwtMiddlewareRoutes.GET("/api/admin/groups/:id", rbacHandler.GetGroup)
		jwtMiddlewareRoutes.POST("/api/admin/groups", rbacHandler.CreateGroup)
		jwtMiddlewareRoutes.PUT("/api/admin/groups/:id", rbacHandler.UpdateGroup)
		jwtMiddlewareRoutes.DELETE("/api/admin/groups/:id", rbacHandler.DeleteGroup)

		jwtMiddlewareRoutes.GET("/api/admin/organizations", rbacHandler.GetOrganizations)
		jwtMiddlewareRoutes.GET("/api/admin/organizations/:id", rbacHandler.GetOrganization)
		jwtMiddlewareRoutes.POST("/api/admin/organizations", rbacHandler.CreateOrganization)
		jwtMiddlewareRoutes.PUT("/api/admin/organizations/:id", rbacHandler.UpdateOrganization)
		jwtMiddlewareRoutes.DELETE("/api/admin/organizations/:id", rbacHandler.DeleteOrganization)

		jwtMiddlewareRoutes.GET("/api/admin/clients", rbacHandler.GetClients)
		jwtMiddlewareRoutes.GET("/api/admin/clients/:id", rbacHandler.GetClient)
		jwtMiddlewareRoutes.POST("/api/admin/clients", rbacHandler.CreateClient)
		jwtMiddlewareRoutes.PUT("/api/admin/clients/:id", rbacHandler.UpdateClient)
		jwtMiddlewareRoutes.DELETE("/api/admin/clients/:id", rbacHandler.DeleteClient)

		jwtMiddlewareRoutes.GET("/api/admin/integrations", rbacHandler.GetIntegrations)
		jwtMiddlewareRoutes.PUT("/api/admin/integrations/github/", rbacHandler.UpdateGithubInstallation)
		jwtMiddlewareRoutes.PUT("/api/admin/integrations/bitbucket/", rbacHandler.UpdateBitbucketInstallation)

		// catalog routes
		jwtMiddlewareRoutes.GET("/api/catalog/entity-labels", catalogHandler.GetCatalogEntityLabels)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-parent-keys", catalogHandler.GetCatalogEntityParentKeys)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-parent-values", catalogHandler.GetCatalogEntityParentValues)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-keys", catalogHandler.GetCatalogEntityKeys)
		jwtMiddlewareRoutes.GET("/api/catalog/entity-values", catalogHandler.GetCatalogEntityValues)
		jwtMiddlewareRoutes.GET("/api/catalog/entities", catalogHandler.GetCatalogEntities)
		jwtMiddlewareRoutes.GET("/api/catalog/entities/:id", catalogHandler.GetCatalogEntity)
		jwtMiddlewareRoutes.POST("/api/catalog/entities", catalogHandler.CreateCatalogEntity)
		jwtMiddlewareRoutes.PUT("/api/catalog/entities/:id", catalogHandler.UpdateCatalogEntity)
		jwtMiddlewareRoutes.DELETE("/api/catalog/entities/:id", catalogHandler.DeleteCatalogEntity)
		jwtMiddlewareRoutes.GET("/api/catalog/users", catalogHandler.GetCatalogUsers)
		jwtMiddlewareRoutes.GET("/api/catalog/users/:id", catalogHandler.GetCatalogUser)
		jwtMiddlewareRoutes.GET("/api/catalog/groups", catalogHandler.GetCatalogGroups)
		jwtMiddlewareRoutes.GET("/api/catalog/groups/:id", catalogHandler.GetCatalogGroup)

		// do not require claims
		jwtMiddlewareRoutes.GET("/api/config", ziplineeHandler.GetConfig)
		jwtMiddlewareRoutes.GET("/api/config/credentials", ziplineeHandler.GetConfigCredentials)
		jwtMiddlewareRoutes.GET("/api/config/trustedimages", ziplineeHandler.GetConfigTrustedImages)
		jwtMiddlewareRoutes.GET("/api/config/buildControl", ziplineeHandler.GetConfigBuildControl)
		jwtMiddlewareRoutes.GET("/api/pipelines", ziplineeHandler.GetPipelines)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo", ziplineeHandler.GetPipeline)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/recentbuilds", ziplineeHandler.GetPipelineRecentBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/buildbranches", ziplineeHandler.GetPipelineBuildBranches)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds", ziplineeHandler.GetPipelineBuilds)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId", ziplineeHandler.GetPipelineBuild)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/warnings", ziplineeHandler.GetPipelineBuildWarnings)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/alllogs", ziplineeHandler.GetPipelineBuildLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases", ziplineeHandler.GetPipelineReleases)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId", ziplineeHandler.GetPipelineRelease)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/alllogs", ziplineeHandler.GetPipelineReleaseLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/botnames", ziplineeHandler.GetPipelineBotNames)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots", ziplineeHandler.GetPipelineBots)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId", ziplineeHandler.GetPipelineBot)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/alllogs", ziplineeHandler.GetPipelineBotLogsPerPage)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsdurations", ziplineeHandler.GetPipelineStatsBuildsDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesdurations", ziplineeHandler.GetPipelineStatsReleasesDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botsdurations", ziplineeHandler.GetPipelineStatsBotsDurations)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildscpu", ziplineeHandler.GetPipelineStatsBuildsCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasescpu", ziplineeHandler.GetPipelineStatsReleasesCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botscpu", ziplineeHandler.GetPipelineStatsBotsCPUUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/buildsmemory", ziplineeHandler.GetPipelineStatsBuildsMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/releasesmemory", ziplineeHandler.GetPipelineStatsReleasesMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/stats/botsmemory", ziplineeHandler.GetPipelineStatsBotsMemoryUsageMeasurements)
		jwtMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/warnings", ziplineeHandler.GetPipelineWarnings)
		jwtMiddlewareRoutes.GET("/api/builds", ziplineeHandler.GetAllPipelineBuilds)
		jwtMiddlewareRoutes.GET("/api/releases", ziplineeHandler.GetAllPipelineReleases)
		jwtMiddlewareRoutes.GET("/api/bots", ziplineeHandler.GetAllPipelineBots)
		jwtMiddlewareRoutes.GET("/api/notifications", ziplineeHandler.GetAllNotifications)
		jwtMiddlewareRoutes.GET("/api/releasetargets/pipelines", ziplineeHandler.GetAllPipelinesReleaseTargets)
		jwtMiddlewareRoutes.GET("/api/releasetargets/releases", ziplineeHandler.GetAllReleasesReleaseTargets)
		jwtMiddlewareRoutes.GET("/api/catalog/filters", ziplineeHandler.GetCatalogFilters)
		jwtMiddlewareRoutes.GET("/api/catalog/filtervalues", ziplineeHandler.GetCatalogFilterValues)
		jwtMiddlewareRoutes.GET("/api/stats/pipelinescount", ziplineeHandler.GetStatsPipelinesCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildscount", ziplineeHandler.GetStatsBuildsCount)
		jwtMiddlewareRoutes.GET("/api/stats/releasescount", ziplineeHandler.GetStatsReleasesCount)
		jwtMiddlewareRoutes.GET("/api/stats/botscount", ziplineeHandler.GetStatsBotsCount)
		jwtMiddlewareRoutes.GET("/api/stats/buildsduration", ziplineeHandler.GetStatsBuildsDuration)
		jwtMiddlewareRoutes.GET("/api/stats/buildsadoption", ziplineeHandler.GetStatsBuildsAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/releasesadoption", ziplineeHandler.GetStatsReleasesAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/botsadoption", ziplineeHandler.GetStatsBotsAdoption)
		jwtMiddlewareRoutes.GET("/api/stats/mostbuilds", ziplineeHandler.GetStatsMostBuilds)
		jwtMiddlewareRoutes.GET("/api/stats/mostreleases", ziplineeHandler.GetStatsMostReleases)
		jwtMiddlewareRoutes.GET("/api/stats/mostbots", ziplineeHandler.GetStatsMostBots)
		jwtMiddlewareRoutes.GET("/api/manifest/templates", ziplineeHandler.GetManifestTemplates)
		jwtMiddlewareRoutes.POST("/api/manifest/generate", ziplineeHandler.GenerateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/validate", ziplineeHandler.ValidateManifest)
		jwtMiddlewareRoutes.POST("/api/manifest/encrypt", ziplineeHandler.EncryptSecret)
		jwtMiddlewareRoutes.GET("/api/labels/frequent", ziplineeHandler.GetFrequentLabels)

		// communication from build/release jobs back to api
		jwtMiddlewareRoutes.POST("/api/commands", ziplineeHandler.Commands)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", ziplineeHandler.PostPipelineBuildLogs)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs", ziplineeHandler.PostPipelineReleaseLogs)
		jwtMiddlewareRoutes.POST("/api/pipelines/:source/:owner/:repo/bots/:botId/logs", ziplineeHandler.PostPipelineBotLogs)

		// do not require claims and avoid re-zipping
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs", ziplineeHandler.GetPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs/tail", ziplineeHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logsbyid/:id", ziplineeHandler.GetPipelineBuildLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/builds/:revisionOrId/logs.stream", ziplineeHandler.TailPipelineBuildLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs", ziplineeHandler.GetPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs/tail", ziplineeHandler.TailPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logsbyid/:id", ziplineeHandler.GetPipelineReleaseLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/releases/:releaseId/logs.stream", ziplineeHandler.TailPipelineReleaseLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs", ziplineeHandler.GetPipelineBotLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs/tail", ziplineeHandler.TailPipelineBotLogs)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logsbyid/:id", ziplineeHandler.GetPipelineBotLogsByID)
		preZippedJWTMiddlewareRoutes.GET("/api/pipelines/:source/:owner/:repo/bots/:botId/logs.stream", ziplineeHandler.TailPipelineBotLogs)
	}

	// default routes
	routes.GET("/liveness", func(c *gin.Context) {
		c.String(http.StatusOK, "I'm alive!")
	})
	routes.GET("/readiness", func(c *gin.Context) {
		res, err := gcsMigratorHealthClient.Check(context.Background(), &migrationpb.HealthCheckRequest{})
		if err != nil {
			log.Warn().Err(err).Msg("error checking readiness of gcs-migrator")
			c.String(http.StatusInternalServerError, "failed to check status of gcs-migrator")
			return
		}
		if res.Status == migrationpb.HealthCheckResponse_SERVING {
			c.String(http.StatusOK, "I'm ready!")
			return
		}
		log.Warn().Int32("gcsMigratorStatus", int32(res.Status)).Err(err).Msg("error checking status of gcs-migrator")
		c.String(http.StatusBadGateway, "invalid status from gcs-migrator")
	})
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": http.StatusText(http.StatusNotFound), "message": "Page not found"})
	})

	// instantiate servers instead of using router.Run in order to handle graceful shutdown
	log.Debug().Msg("Starting server...")
	srv := &http.Server{
		Addr:        *apiAddress,
		Handler:     router,
		ReadTimeout: 30 * time.Second,
		//WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Debug().Msg("Listening for incoming requests...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Starting gin router failed")
		}
	}()

	return srv
}

// initJaeger returns an instance of Jaeger Tracer that can be configured with environment variables
// https://github.com/jaegertracing/jaeger-client-go#environment-variables
func initJaeger() io.Closer {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger config from environment variables failed")
	}

	// disable jaeger if service name is empty
	if cfg.ServiceName == "" {
		cfg.Disabled = true
	}

	closer, err := cfg.InitGlobalTracer(cfg.ServiceName, jaegercfg.Logger(jaeger.StdLogger), jaegercfg.Metrics(jprom.New()))

	if err != nil {
		log.Fatal().Err(err).Msg("Generating Jaeger tracer failed")
	}

	return closer
}

// start gcs-migrator
func startGcsMigrator() {
	if gcsMigrator != nil {
		if gcsMigratorStartedAt.Add(gcsMigratorRestartDelay).After(time.Now()) {
			log.Debug().Msg("gcs-migrator was started less than a minutes ago, not restarting it")
			return
		}
		log.Warn().Msg("Restarting gcs-migrator...")
		err := gcsMigrator.Process.Kill()
		if err != nil {
			log.Error().Err(err).Msg("failed to stop gcs-migrator")
		}
	}
	gcsMigrator = exec.Command("/gcs-migrator")
	gcsMigrator.Stdout = os.Stdout
	gcsMigrator.Stderr = os.Stderr
	err := gcsMigrator.Start()
	if err != nil {
		log.Warn().Err(err).Msg("Starting gcs-migrator failed")
	}
	gcsMigratorStartedAt = time.Now()
}
