package pubsubapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	manifest "github.com/ziplineeci/ziplinee-ci-manifest"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "pubsubapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) SubscriptionForTopic(ctx context.Context, message PubSubPushMessage) (event *manifest.ZiplineePubSubEvent, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "SubscriptionForTopic"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.SubscriptionForTopic(ctx, message)
}

func (c *tracingClient) SubscribeToTopic(ctx context.Context, projectID, topicID string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "SubscribeToTopic"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToTopic(ctx, projectID, topicID)
}

func (c *tracingClient) SubscribeToPubsubTriggers(ctx context.Context, manifestString string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "SubscribeToPubsubTriggers"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.SubscribeToPubsubTriggers(ctx, manifestString)
}
