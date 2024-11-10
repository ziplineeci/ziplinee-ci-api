package cloudsourceapi

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
)

// NewMetricsClient returns a new instance of a metrics Client.
func NewMetricsClient(c Client, requestCount metrics.Counter, requestLatency metrics.Histogram) Client {
	return &metricsClient{c, requestCount, requestLatency}
}

type metricsClient struct {
	Client         Client
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
}

func (c *metricsClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetAccessToken", begin)
	}(time.Now())

	return c.Client.GetAccessToken(ctx)
}

func (c *metricsClient) GetZiplineeManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetZiplineeManifest", begin)
	}(time.Now())

	return c.Client.GetZiplineeManifest(ctx, accesstoken, notification, gitClone)
}

func (c *metricsClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	defer func(begin time.Time) { api.UpdateMetrics(c.requestCount, c.requestLatency, "JobVarsFunc", begin) }(time.Now())

	return c.Client.JobVarsFunc(ctx)
}
