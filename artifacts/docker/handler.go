package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/harness/artifacts-cgi/common"
	"github.com/sirupsen/logrus"
)

type AuthType string
type ProviderType string

var (
	AuthTypeUsernamePassword AuthType = "UsernamePassword"
	AuthTypeAnonymous        AuthType = "Anonymous"

	ProviderTypeDockerHub ProviderType = "DockerHub"
)

type DockerArtifactParams struct {
	Url          string       `json:"url"`
	ProviderType ProviderType `json:"provider_type"`
	AuthType     AuthType     `json:"auth_type"`
	Username     string       `json:"username"`
	Password     string       `json:"password"`
}

type DockerHandler struct {
	params *DockerArtifactParams
	client *DockerClient
}

func New(artifactParams json.RawMessage) (*DockerHandler, error) {
	var params DockerArtifactParams
	if err := json.Unmarshal(artifactParams, &params); err != nil {
		return nil, fmt.Errorf("failed to decode artifactParams into DockerArtifactParams with error=[%w]", err)
	}
	err := validateParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid DockerTriggerParams: %w", err)
	}
	client := NewDockerClient(params.Url, params.Username, params.Password)
	return &DockerHandler{params: &params, client: client}, nil
}

func validateParams(params DockerArtifactParams) error {
	if params.Url == "" {
		return fmt.Errorf("url is empty")
	}
	if params.ProviderType == "" {
		return fmt.Errorf("providerType is empty")
	}
	if params.AuthType == "" {
		return fmt.Errorf("authType is empty")
	}
	if params.AuthType == AuthTypeUsernamePassword {
		if params.Username == "" {
			return fmt.Errorf("username is empty")
		}
		if params.Password == "" {
			return fmt.Errorf("password is empty")
		}
	}
	return nil

}

func (d *DockerHandler) Validate() (*common.ValidationResponse, error) {
	switch providerType := d.params.ProviderType; providerType {
	case ProviderTypeDockerHub:
		err := d.client.Validate(context.Background())
		if err != nil {
			logrus.WithError(err).Errorf("failed validating artifact server")
			return &common.ValidationResponse{Status: common.ValidationStatusFailure, Error: common.ErrorDetail{Message: err.Error()}}, nil
		}
		logrus.Infof("successfully validated artifact server")
		return &common.ValidationResponse{Status: common.ValidationStatusSuccess}, nil
	default:
		return nil, fmt.Errorf("Unsupport docker provider type [%s]", providerType)
	}
}
