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
    return err.Message
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
        Message: fmt.Sprintf("%d %s", code, http.StatusText(code)),
    }
}

func New(code int, message string) *Error {
    return &Error{
        Code: code,
        Message: fmt.Sprintf("%d %s\n%s", code, http.StatusText(code), message),
    }
}
