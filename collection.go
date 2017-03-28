package rest

import (
    "encoding/json"
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io/ioutil"
    "net/http"
    "net/url"
    "path"
    "reflect"
    "strconv"
    "strings"
)

var (
    ErrMethodNotAllowed = rest_error.NewByCode(http.StatusMethodNotAllowed)
)

type Request struct {
    IDs   []string
    Query url.Values
}

type ResourceHandler interface {
    EmptyItem() interface{}
    Create(request *Request, item interface{}) (string, error)
    Read(request *Request) (interface{}, error)
    List(request *Request) (interface{}, error)
    Update(request *Request, item interface{}) error
    Replace(request *Request, item interface{}) error
    Delete(request *Request) error
    BatchDelete(request *Request) error
}

type Collection struct {
    level          int
    handler        ResourceHandler
    subCollections map[string]*Collection
}

func newCollection(level int, handler ResourceHandler) *Collection {
    return &Collection{
        level: level,
        handler: handler,
        subCollections: make(map[string]*Collection),
    }
}

func (server *Server) Collection(name string, handler ResourceHandler) *Collection {
    collection := newCollection(0, handler)

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

        restRequest := &Request{
            IDs:   ids,
            Query: httpRequest.URL.Query(),
        }

        actualCollection.handle(restRequest, response, httpRequest)
    }

    server.mux.HandleFunc(collectionPath, handlerFunc)
    server.mux.HandleFunc(fmt.Sprintf("%s/", collectionPath), handlerFunc)

    return collection
}

func (collection *Collection) SubCollection(name string, handler ResourceHandler) *Collection {
    subCollection := newCollection(collection.level + 1, handler)
    collection.subCollections[name] = subCollection
    return subCollection
}

func splitPath(path string) []string {
    return strings.FieldsFunc(strings.TrimSpace(path), func(r rune) bool {
        return r == '/'
    })
}

func (collection *Collection) handle(restRequest *Request, response http.ResponseWriter, httpRequest *http.Request) {
    collectionRequest := collection.level == len(restRequest.IDs)

    switch strings.ToUpper(httpRequest.Method) {
    case "GET":
        if collectionRequest {
            collection.handleList(restRequest, response)
            return
        } else {
            collection.handleRead(restRequest, response)
            return
        }

    case "POST":
        if collectionRequest {
            collection.handleCreate(restRequest, response, httpRequest)
            return
        } else {
            collection.handleUpdate(restRequest, response, httpRequest)
            return
        }

    case "PUT":
        if !collectionRequest {
            collection.handleReplace(restRequest, response, httpRequest)
            return
        }

    case "DELETE":
        if collectionRequest {
            collection.handleBatchDelete(restRequest, response)
            return
        } else {
            collection.handleDelete(restRequest, response)
            return
        }
    }

    writeError(response, ErrMethodNotAllowed)
}

func (collection *Collection) handleList(restRequest *Request, response http.ResponseWriter) {
    items, err := collection.handler.List(restRequest)
    if err != nil {
        writeError(response, err)
        return
    }

    itemsJson, err := marshal(items, restRequest)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemsJson)
}

func (collection *Collection) handleRead(restRequest *Request, response http.ResponseWriter) {
    item, err := collection.handler.Read(restRequest)
    if err != nil {
        writeError(response, err)
        return
    }

    if isNil(item) {
        writeError(response, rest_error.NewByCode(http.StatusNotFound))
        return
    }

    itemJson, err := marshal(item, restRequest)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemJson)
}

func (collection *Collection) handleCreate(restRequest *Request, response http.ResponseWriter, httpRequest *http.Request) {
    item, err := collection.readItem(httpRequest, "create")
    if err != nil {
        writeError(response, err)
        return
    }

    id, err := collection.handler.Create(restRequest, item)
    if err != nil {
        writeError(response, err)
        return
    }

    relativeLocation := path.Clean(fmt.Sprintf("/%s/%s", strings.Trim(httpRequest.URL.Path, "/"), id))

    response.Header().Add("Location", relativeLocation)

    writeAnswer(response, http.StatusCreated, nil)
}

func (collection *Collection) handleUpdate(restRequest *Request, response http.ResponseWriter, httpRequest *http.Request) {
    item, err := collection.readItem(httpRequest, "update")
    if err != nil {
        writeError(response, err)
        return
    }

    if err := collection.handler.Update(restRequest, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (collection *Collection) handleReplace(restRequest *Request, response http.ResponseWriter, httpRequest *http.Request) {
    item, err := collection.readItem(httpRequest, "replace")
    if err != nil {
        writeError(response, err)
        return
    }

    if err := collection.handler.Replace(restRequest, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (collection *Collection) handleDelete(restRequest *Request, response http.ResponseWriter) {
    if err := collection.handler.Delete(restRequest); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (collection *Collection) handleBatchDelete(restRequest *Request, response http.ResponseWriter) {
    if err := collection.handler.BatchDelete(restRequest); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func isNil(item interface{}) bool {
    if item == nil {
        return true
    }

    value := reflect.ValueOf(item)

    switch value.Kind() {
    case reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Map:
        return value.IsNil()
    default:
        return false
    }
}

func marshal(v interface{}, restRequest *Request) ([]byte, error) {
    if indentParam := restRequest.Query.Get("indent"); indentParam != "" {
        if indentCount, err := strconv.Atoi(indentParam); err == nil && 0 < indentCount && indentCount <= 32 {
            return json.MarshalIndent(v, "", strings.Repeat(" ", indentCount))
        }
    }
    return json.Marshal(v)
}

func (collection *Collection) readItem(httpRequest *http.Request, action string) (interface{}, error) {
    itemJson, err := ioutil.ReadAll(httpRequest.Body)
    if err != nil {
        return nil, err
    }

    item := collection.handler.EmptyItem()

    if err := json.Unmarshal(itemJson, item); err != nil {
        return nil, rest_error.New(http.StatusBadRequest, err.Error())
    }

    if err := checkRestrictions(item, action); err != nil {
        return nil, rest_error.New(http.StatusBadRequest, err.Error())
    }

    return item, nil
}

func writeError(response http.ResponseWriter, err error) {
    code := http.StatusInternalServerError
    if restError, ok := err.(*rest_error.Error); ok {
        code = restError.Code
    }
    http.Error(response, err.Error(), code)
}

func writeAnswer(response http.ResponseWriter, status int, content []byte) {
    if content != nil {
        response.Header().Add("Content-Type", "application/json")
    }

    response.WriteHeader(status)

    if content != nil {
        for len(content) != 0 {
            n, _ := response.Write(content)
            if n == 0 {
                return
            }
            content = content[n:]
        }
    }
}
