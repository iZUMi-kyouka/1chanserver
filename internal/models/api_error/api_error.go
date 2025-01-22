package api_error

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type APIError struct {
	error
	httpStatus int
	message    string
}

func (e *APIError) Unwrap() error {
	return e.error
}

func (e *APIError) HTTPStatus() int {
	return e.httpStatus
}

func (e *APIError) Message() string {
	return e.message
}

func New(e error, httpStatus int, message string) APIError {
	return APIError{
		error:      e,
		httpStatus: httpStatus,
		message:    message,
	}
}

func NewFromErr(e error, httpStatus int) APIError {
	return APIError{
		error:      e,
		httpStatus: httpStatus,
		message:    "",
	}
}

func NewFromStr(s string, httpStatus int) APIError {
	return APIError{
		error:      errors.New(s),
		httpStatus: httpStatus,
		message:    "",
	}
}

func ToResponse(c *gin.Context, e error) {
	var currentErr APIError

	if errors.As(e, &currentErr) {
		if currentErr.Message() == "" {
			c.JSON(currentErr.HTTPStatus(), gin.H{
				"description": currentErr.error.Error()})
		} else {
			c.JSON(currentErr.HTTPStatus(), gin.H{
				"message":     currentErr.Message(),
				"description": currentErr.Error(),
			})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"description": e.Error(),
		})
	}
}
