package rest

import (
    "net/http"
)

type Error struct {
    code    int
    message string
}

func (err *Error) Error() string {
    return err.message
}

var (
    ErrNotFound = &Error{code: http.StatusNotFound}
    ErrMethodNotAllowed = &Error{code: http.StatusMethodNotAllowed}
)
