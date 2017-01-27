package rest

import (
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io"
    "net/http"
    "os"
    "strings"
)

func (server *Server) Static(pattern, contentType, rootFolder string) {
    path := server.path(pattern)
    if !strings.HasSuffix(path, "/") {
        path = fmt.Sprintf("%s/", path)
    }
    
    server.mux.HandleFunc(path, func(response http.ResponseWriter, request *http.Request) {
        filePath := fmt.Sprintf("%s/%s", rootFolder, strings.TrimPrefix(strings.TrimSpace(request.URL.Path), path))
        WriteFile(response, contentType, filePath)
    })
}

func WriteFile(response http.ResponseWriter, contentType string, filePath string) {
    file, err := os.Open(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            rest_error.NewByCode(http.StatusNotFound).Send(response)
        } else if os.IsPermission(err) {
            rest_error.NewByCode(http.StatusForbidden).Send(response)
        } else {
            rest_error.New(http.StatusInternalServerError, err.Error()).Send(response)
        }
        return
    }

    defer file.Close()

    fileStat, err := file.Stat()
    if err != nil {
        rest_error.New(http.StatusInternalServerError, err.Error()).Send(response)
        return
    } else if fileStat.IsDir() {
        rest_error.NewByCode(http.StatusForbidden).Send(response)
        return
    }

    response.Header().Add("Content-Type", contentType)

    io.Copy(response, file)
}
