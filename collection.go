package rest

import (
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "net/http"
    "strings"
)

var (
    ErrMethodNotAllowed = rest_error.NewByCode(http.StatusMethodNotAllowed)
)

type Request struct {
    *http.Request

    Level int
    IDs   []string
}

type Handler interface {
    ServeHTTP(request *Request, response http.ResponseWriter)
}

type HandlerFunc func(*Request, http.ResponseWriter)

func (f HandlerFunc) ServeHTTP(request *Request, response http.ResponseWriter) {
    f(request, response)
}

type Collection struct {
    level          int
    handler        Handler
    subCollections map[string]*Collection
}

func newCollection(level int) *Collection {
    return &Collection{
        level:          level,
        subCollections: make(map[string]*Collection),
    }
}

func (server *Server) Collection(name string) *Collection {
    collection := newCollection(0)

    if strings.Contains(name, "/") {
        panic(fmt.Sprintf("Slash in collection name: %s", name))
    }

    if len(name) == 0 {
        panic("Empty collection name")
    }

    collectionPath := server.path(fmt.Sprintf("/%s", name))
    collectionIndex := len(splitPath(collectionPath)) - 1

    handlerFunc := func(response http.ResponseWriter, httpRequest *http.Request) {
        pathNames := splitPath(httpRequest.URL.Path)[collectionIndex:]

        if len(pathNames) == 0 { // cannot happen
            writeError(response, fmt.Errorf("Invalid URL path: %s", httpRequest.URL.Path))
            return
        }

        ids := make([]string, 0)
        actualCollection := collection
        id := true

        for i, pathName := range pathNames[1:] {
            if id {
                ids = append(ids, pathName)
                id = false
            } else {
                actualCollection = actualCollection.subCollections[pathName]
                if actualCollection == nil {
                    writeError(response, rest_error.New(http.StatusNotFound, fmt.Sprintf("Path is not found: /%s", strings.Join(pathNames[:i + 2], "/"))))
                    return
                }
                id = true
            }
        }

        if actualCollection.handler == nil {
            writeError(response, ErrMethodNotAllowed)
            return
        }

        request := &Request{
            Request: httpRequest,
            Level:   actualCollection.level,
            IDs:     ids,
        }

        actualCollection.handler.ServeHTTP(request, response)
    }

    server.mux.HandleFunc(collectionPath, handlerFunc)
    server.mux.HandleFunc(fmt.Sprintf("%s/", collectionPath), handlerFunc)

    return collection
}

func (collection *Collection) CustomHandler(handler Handler) *Collection {
    collection.handler = handler
    return collection
}

func (collection *Collection) CustomHandlerFunc(handlerFunc func(*Request, http.ResponseWriter)) *Collection {
    return collection.CustomHandler(HandlerFunc(handlerFunc))
}

func (collection *Collection) SubCollection(name string) *Collection {
    subCollection := newCollection(collection.level + 1)
    collection.subCollections[name] = subCollection
    return subCollection
}

func splitPath(path string) []string {
    return strings.FieldsFunc(strings.TrimSpace(path), func(r rune) bool {
        return r == '/'
    })
}

func writeError(response http.ResponseWriter, err error) {
    var code int
    var message string

    if restError, ok := err.(*rest_error.Error); ok {
        code = restError.Code
        message = restError.Message
    } else {
        code = http.StatusInternalServerError
        message = err.Error()
    }

    http.Error(response, message, code)
}
