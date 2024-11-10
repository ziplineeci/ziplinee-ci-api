package cloudsourceapi

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
)

// NewTracingClient returns a new instance of a tracing Client.
func NewTracingClient(c Client) Client {
	return &tracingClient{c, "cloudsourceapi"}
}

type tracingClient struct {
	Client Client
	prefix string
}

func (c *tracingClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetAccessToken"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetAccessToken(ctx)
}

func (c *tracingClient) GetZiplineeManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "GetZiplineeManifest"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return c.Client.GetZiplineeManifest(ctx, accesstoken, notification, gitClone)
}

func (c *tracingClient) JobVarsFunc(ctx context.Context) func(context.Context, string, string, string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(c.prefix, "JobVarsFunc"))
	defer func() { api.FinishSpan(span) }()

	return c.Client.JobVarsFunc(ctx)
}
