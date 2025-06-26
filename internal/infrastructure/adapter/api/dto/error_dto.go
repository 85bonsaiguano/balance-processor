package dto

// ErrorResponse represents a standardized error response for the API
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
