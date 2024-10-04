package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cgi"

	"github.com/harness/artifacts-cgi/artifacts/docker"
	"github.com/harness/artifacts-cgi/common"
)

func main() {
	http.HandleFunc("/", handle)
	cgi.Serve(http.DefaultServeMux)
}

type ArtifactType string
type ArtifactOperation string

var (
	Docker ArtifactType = "DockerRegistry"

	Validate ArtifactOperation = "VALIDATE"
)

type Params struct {
	ArtifactType      ArtifactType      `json:"artifact_type"`
	ArtifactOperation ArtifactOperation `json:"artifact_operation"`
	ArtifactParams    json.RawMessage   `json:"artifact_params"`
}

func handle(w http.ResponseWriter, r *http.Request) {
	// unmarshal the input
	params, err := parseParams(r)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// get the artifact handler
	handler, err := getHandler(params)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := applyOperation(handler, params)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	sendSuccessResponse(w, res)
}

func applyOperation(handler common.ArtifactHandler, params *Params) (interface{}, error) {
	var res interface{}
	var err error
	switch operation := params.ArtifactOperation; operation {
	case Validate:
		res, err = handler.Validate()
	default:
		err = fmt.Errorf("unsupported artifact operation [%s]", operation)
	}
	return res, err
}

func getHandler(params *Params) (common.ArtifactHandler, error) {
	var handler common.ArtifactHandler
	var err error
	switch artifactType := params.ArtifactType; artifactType {
	case Docker:
		handler, err = docker.New(params.ArtifactParams)
	default:
		err = fmt.Errorf("unsupported artifact type [%s]", artifactType)
	}
	return handler, err
}

func parseParams(r *http.Request) (*Params, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read request body")
	}
	defer r.Body.Close()

	params := new(Params)
	if err := json.Unmarshal(body, params); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}
	return params, nil
}

func sendSuccessResponse(w http.ResponseWriter, response interface{}) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func sendErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": errMsg,
	})
}
