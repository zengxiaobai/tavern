package errors

import (
	"fmt"
	"net/http"
)

type Error struct {
	Code    int
	Headers http.Header
	cause   error
}

func New(code int, headers http.Header) *Error {
	return &Error{
		Code:    code,
		Headers: headers,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("error: code = %d headers = %v cause = %v", e.Code, e.Headers, e.cause)
}

func (e *Error) WithCause(err error) *Error {
	e.cause = err
	return e
}
