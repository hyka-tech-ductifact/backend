package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// StrictBindJSON decodes the request body into obj, rejecting unknown fields.
// It combines json.Decoder.DisallowUnknownFields with Gin's struct validation,
// enforcing additionalProperties: false at the API level.
func StrictBindJSON(c *gin.Context, obj any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	defer io.Copy(io.Discard, c.Request.Body) //nolint:errcheck

	if err := dec.Decode(obj); err != nil {
		return formatJSONError(err)
	}

	if err := binding.Validator.ValidateStruct(obj); err != nil {
		return err
	}

	return nil
}

// formatJSONError converts json.Decoder errors into user-friendly messages.
func formatJSONError(err error) error {
	msg := err.Error()
	// json: unknown field "xxx"
	if strings.HasPrefix(msg, "json: unknown field") {
		return fmt.Errorf("unknown field in request body: %s", strings.TrimPrefix(msg, "json: unknown field "))
	}
	return err
}

// IsValidationError checks whether the error came from struct validation (go-playground/validator).
func IsValidationError(err error) bool {
	_, ok := err.(validator.ValidationErrors)
	return ok
}
