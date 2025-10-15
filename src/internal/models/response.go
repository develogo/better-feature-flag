package models

type FlagsResponse struct {
	Flags map[string]interface{} `json:"flags"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
