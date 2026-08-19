package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"git.frontiir.net/sa-dev/frontiirgopackages/pkg/frontlog"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RestErr represents a structured REST API error response
type RestErr struct {
	Status  int    `json:"status"`         // HTTP status code
	Message string `json:"message"`        // User-friendly error message
	Code    string `json:"code,omitempty"` // Machine-readable business error code (e.g. "SLA_EXPIRED")
	Data    any    `json:"data,omitempty"` // Optional structured payload (e.g. conflicting ticket detail)
}

func (req RestErr) IsRequest() bool {
	return true
}

type ArrayRestErr struct {
	Message []string `json:"message"`
	Status  int      `json:"status"`
	Error   string   `json:"error"`
}

func (req ArrayRestErr) IsRequest() bool {
	return true
}

func BadRequest(message string) *RestErr {
	return &RestErr{
		Message: message,
		Status:  400,
	}
}

func InputErr(message string) *RestErr {
	return &RestErr{
		Message: message,
		Status:  422,
	}
}

func ValidationErr(message []string) *ArrayRestErr {
	return &ArrayRestErr{
		Message: message,
		Status:  422,
		Error:   "unprocessable entity",
	}
}

func SuccessMessage(message string) *RestErr {
	return &RestErr{
		Message: message,
		Status:  200,
	}
}

func StringToInt32(x string) (int32, error) {
	i, err := strconv.ParseInt(x, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}

func StringToUint(x string) (uint, error) {
	i, err := strconv.ParseInt(x, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(i), nil
}

// LogError logs an error using structured logging (no longer fatal)
func LogError(err error) {
	if err != nil {
		frontlog.Logger.Error("Error occurred", zap.Error(err))
	}
}

// Error implements the error interface for RestErr
func (e *RestErr) Error() string {
	return fmt.Sprintf("Status: %d, Message: %s", e.Status, e.Message)
}

// BadRequestErr creates a 400 Bad Request error response
// Usage:
// return utils.BadRequestErr("Invalid email format: %s", email)
func BadRequestErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusBadRequest,
		Message: fmt.Sprintf(format, args...),
	}
}

// Helper functions for other common error types
func InternalErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusInternalServerError,
		Message: fmt.Sprintf(format, args...),
	}
}

func NotFound(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusNotFound,
		Message: fmt.Sprintf(format, args...),
	}
}

// ConflictErr creates a 409 Conflict error response
func ConflictErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusConflict,
		Message: fmt.Sprintf(format, args...),
	}
}

// ForbiddenErr creates a 403 Forbidden error response
func ForbiddenErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusForbidden,
		Message: fmt.Sprintf(format, args...),
	}
}

// BadGatewayErr creates a 502 Bad Gateway error response
func BadGatewayErr(format string, args ...interface{}) *RestErr {
	return &RestErr{
		Status:  http.StatusBadGateway,
		Message: fmt.Sprintf(format, args...),
	}
}

func ErrResponse(result error) *RestErr {

	if result != nil {
		// Get caller information for better context
		pc, file, line, _ := runtime.Caller(1)
		funcName := runtime.FuncForPC(pc).Name()

		// Structured logging
		log.Printf("[ERROR] %s:%d %s - Error: %v\n", file, line, funcName, result)

		// Handle specific error types
		switch {
		case errors.Is(result, gorm.ErrRecordNotFound):
			log.Printf("[WARN] Record not found: %v", result)
			return NotFound("Requested resource not found")

		case strings.Contains(result.Error(), "record not found"), strings.Contains(strings.ToLower(result.Error()), "customfield does not exist"):
			log.Printf("[WARN] Record not found (generic): %v", result)
			return NotFound("%s", result.Error())

		default:
			log.Printf("[ERROR] Internal server error: %v", result)
			return InternalErr("Internal server error")
		}
	}

	// Log unexpected empty error
	log.Printf("[WARN] ErrResponse called with nil error")
	return &RestErr{
		Status:  http.StatusInternalServerError,
		Message: "Unexpected error occurred",
	}
}

func FormatJSON(message string, v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}

	jsonString := string(b)
	if message != "" {
		return fmt.Sprintf("%s:\n%s", message, jsonString)
	}
	return jsonString
}

func GetEnvValue(key string) string {
	return os.Getenv(key)
}

// GetEnvBool retrieves a boolean environment variable
func GetEnvBool(key string) bool {
	boolValue, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		log.Printf("Error parsing boolean environment variable %s: %v, defaulting to false", key, err)
		return false
	}
	return boolValue
}

// ToJSONString converts an object to a formatted JSON string without logging
// Useful for logging with zap when you want the raw JSON string
func ToJSONString(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling: %v", err)
	}
	return string(b)
}
