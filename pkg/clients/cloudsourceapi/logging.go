package cloudsourceapi

import (
	"context"

	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
)

// NewLoggingClient returns a new instance of a logging Client.
func NewLoggingClient(c Client) Client {
	return &loggingClient{c, "cloudsourceapi"}
}

type loggingClient struct {
	Client Client
	prefix string
}

func (c *loggingClient) GetAccessToken(ctx context.Context) (accesstoken AccessToken, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetAccessToken", err) }()

	return c.Client.GetAccessToken(ctx)
}

func (c *loggingClient) GetZiplineeManifest(ctx context.Context, accesstoken AccessToken, notification PubSubNotification, gitClone func(string, string, string) error) (valid bool, manifest string, err error) {
	defer func() { api.HandleLogError(c.prefix, "Client", "GetZiplineeManifest", err) }()

	return c.Client.GetZiplineeManifest(ctx, accesstoken, notification, gitClone)
}

func (c *loggingClient) JobVarsFunc(ctx context.Context) func(ctx context.Context, repoSource, repoOwner, repoName string) (token string, err error) {
	return c.Client.JobVarsFunc(ctx)
}
