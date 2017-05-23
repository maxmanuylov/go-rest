package rest

import (
    "bufio"
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

func (server *Server) Static(pattern, contentType, folderPath string) {
    server.doStatic(pattern, folderPath, func(_ string) string {
        return contentType
    })
}

func (server *Server) StaticExt(pattern, folderPath string, ext2contentType map[string]string) {
    server.doStatic(pattern, folderPath, func(filePath string) string {
        return ext2contentType[filepath.Ext(filePath)]
    })
}

func (server *Server) doStatic(pattern, folderPath string, contentTypeFunc func(string) string) {
    path := server.path(pattern)
    if !strings.HasSuffix(path, "/") {
        path = fmt.Sprintf("%s/", path)
    }

    cleanFolderPath := filepath.Clean(folderPath)

    server.mux.HandleFunc(path, func(response http.ResponseWriter, request *http.Request) {
        filePath := filepath.Join(cleanFolderPath, strings.TrimPrefix(strings.TrimSpace(request.URL.Path), path))
        WriteFile(response, contentTypeFunc(filePath), filepath.Clean(filePath))
    })
}

func WriteFile(response http.ResponseWriter, contentType, filePath string) {
    doWriteFile(response, contentType, filePath, func(file *os.File) {
        io.Copy(response, file)
    })
}

func WriteTemplate(response http.ResponseWriter, contentType, templateFilePath string, replacements map[string]string) {
    doWriteFile(response, contentType, templateFilePath, func(file *os.File) {
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            response.Write(apply(scanner.Text(), replacements))
        }
    })
}

func apply(text string, replacements map[string]string) []byte {
    result := text
    for key, value := range replacements {
        result = strings.Replace(result, key, value, -1)
    }
    return []byte(result)
}

func doWriteFile(response http.ResponseWriter, contentType, filePath string, writeFunc func(file *os.File)) {
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

    if contentType != "" {
        response.Header().Add("Content-Type", contentType)
    }

    writeFunc(file)
}