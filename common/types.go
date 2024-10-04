package common

type ValidationStatus string

var (
	ValidationStatusSuccess ValidationStatus = "SUCCESS"
	ValidationStatusFailure ValidationStatus = "FAILURE"
)

type ArtifactHandler interface {
	Validate() (*ValidationResponse, error)
}

type ValidationResponse struct {
	Status ValidationStatus `json:"status"`
	Error  ErrorDetail      `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Code    int    `json:"int"`
}
