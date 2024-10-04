package docker

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/harness/artifacts-cgi/common"
)

const (
	apiVersionEndpoint = "/v2/"
)

var (
	timeout    = 10 * time.Second
	retryTimes = 5
)

type DockerClient struct {
	common.HTTPClient
	username string
	password string
}

func NewDockerClient(url, username, password string) *DockerClient {
	return &DockerClient{
		HTTPClient: *common.NewHttpClient(url, false),
		username:   username,
		password:   password,
	}
}

func (d *DockerClient) Validate(ctx context.Context) error {
	_, err := d.Retry(ctx, apiVersionEndpoint, "GET", nil, nil, d.getHeaders(), d.CreateBackoff(ctx, timeout), false, retryTimes)
	return err
}

func (d *DockerClient) getHeaders() map[string]string {
	headers := make(map[string]string)
	auth := d.username + ":" + d.password
	headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	return headers
}
