package slackapi

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

func (c *metricsClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	defer func(begin time.Time) {
		api.UpdateMetrics(c.requestCount, c.requestLatency, "GetUserProfile", begin)
	}(time.Now())

	return c.Client.GetUserProfile(ctx, userID)
}
