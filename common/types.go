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
	Status       ValidationStatus `json:"status"`
	Errors       []ErrorDetail    `json:"errors"`
	ErrorSummary string           `json:"error_summary"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Code    int    `json:"code"`
}
