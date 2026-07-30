package api

// ErrorResponse is the JSON body returned for any non-2xx DeployOS API
// response.
type ErrorResponse struct {
	Error string `json:"error"`
}
