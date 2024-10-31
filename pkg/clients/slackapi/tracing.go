package slackapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "slackapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetUserProfile(ctx context.Context, userID string) (profile *UserProfile, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetUserProfile"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetUserProfile(ctx, userID)
}
