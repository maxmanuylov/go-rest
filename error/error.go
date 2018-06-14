package rest_error

import (
    "fmt"
    "net/http"
)

type Error struct {
    Code    int
    Message string
}

func (err *Error) Error() string {
    if err.Message == "" {
        return fmt.Sprintf("%d %s", err.Code, http.StatusText(err.Code))
    }
    return fmt.Sprintf("%d %s: %s", err.Code, http.StatusText(err.Code), err.Message)
}

func (err *Error) IsClientError() bool {
    return err.Code / 100 == 4
}

func (err *Error) IsServerError() bool {
    return err.Code / 100 == 5
}

func NewByCode(code int) *Error {
    return &Error{
        Code: code,
    }
}

func New(code int, message string) *Error {
    return &Error{
        Code:    code,
        Message: message,
    }
}

func (err *Error) Send(response http.ResponseWriter) {
    http.Error(response, err.Message, err.Code)
}

func Send(err error, response http.ResponseWriter) {
    if restError, ok := err.(*Error); ok {
        restError.Send(response)
    } else {
        http.Error(response, err.Error(), http.StatusInternalServerError)
    }
}
