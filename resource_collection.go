package rest

import (
    "encoding/json"
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io/ioutil"
    "net/http"
    "path"
    "reflect"
    "strconv"
    "strings"
)

type ItemAction string

const (
    Create  ItemAction = "create"
    Update             = "update"
    Replace            = "replace"
)

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

type ActionHandler interface {
    Do(request *Request) error
}

type resourceHandlerAdapter struct {
    resourceHandler ResourceHandler
    customActions   map[string]ActionHandler
}

type ResourceCollection struct {
    *Collection
    
    resourceHandler *resourceHandlerAdapter
}

func (collection *Collection) Handler(handler ResourceHandler) *ResourceCollection {
    resourceHandler := &resourceHandlerAdapter{
        resourceHandler: handler,
        customActions:   make(map[string]ActionHandler),
    }

    return &ResourceCollection{
        Collection:      collection.CustomHandler(resourceHandler),
        resourceHandler: resourceHandler,
    }
}

func (collection *ResourceCollection) CustomAction(method string, handler ActionHandler) *ResourceCollection {
    collection.resourceHandler.customActions[strings.ToUpper(method)] = handler
    return collection
}

func (r *Request) IsFlagSet(flagName string) bool {
    if q := r.URL.Query(); q != nil {
        if values, ok := q[flagName]; ok && len(values) != 0 {
            return parseFlag(values[0])
        }
    }
    return false
}

func parseFlag(flagStr string) bool {
    if flagStr != "" {
        flag, err := strconv.ParseBool(flagStr)
        return err == nil && flag
    }
    return true
}

func (resourceHandler *resourceHandlerAdapter) ServeHTTP(request *Request, response http.ResponseWriter) {
    collectionRequest := request.Level == len(request.IDs)
    method := strings.ToUpper(request.Method)

    switch method {
    case "GET":
        if collectionRequest {
            resourceHandler.handleList(request, response)
            return
        } else {
            resourceHandler.handleRead(request, response)
            return
        }

    case "POST":
        if collectionRequest {
            resourceHandler.handleCreate(request, response)
            return
        } else {
            resourceHandler.handleUpdate(request, response)
            return
        }

    case "PUT":
        if !collectionRequest {
            resourceHandler.handleReplace(request, response)
            return
        }

    case "DELETE":
        if collectionRequest {
            resourceHandler.handleBatchDelete(request, response)
            return
        } else {
            resourceHandler.handleDelete(request, response)
            return
        }

    default:
        if !collectionRequest {
            if handler := resourceHandler.customActions[method]; handler != nil {
                resourceHandler.handleCustomAction(request, handler, response)
                return
            }
        }
    }

    writeError(response, ErrMethodNotAllowed)
}

func (resourceHandler *resourceHandlerAdapter) handleList(request *Request, response http.ResponseWriter) {
    items, err := resourceHandler.resourceHandler.List(request)
    if err != nil {
        writeError(response, err)
        return
    }

    itemsJson, err := request.Marshal(items)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemsJson)
}

func (resourceHandler *resourceHandlerAdapter) handleRead(request *Request, response http.ResponseWriter) {
    item, err := resourceHandler.resourceHandler.Read(request)
    if err != nil {
        writeError(response, err)
        return
    }

    if isNil(item) {
        writeError(response, rest_error.NewByCode(http.StatusNotFound))
        return
    }

    itemJson, err := request.Marshal(item)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemJson)
}

func (resourceHandler *resourceHandlerAdapter) handleCreate(request *Request, response http.ResponseWriter) {
    item, err := resourceHandler.readItem(request, Create)
    if err != nil {
        writeError(response, err)
        return
    }

    id, err := resourceHandler.resourceHandler.Create(request, item)
    if err != nil {
        writeError(response, err)
        return
    }

    relativeLocation := path.Clean(fmt.Sprintf("/%s/%s", strings.Trim(request.URL.Path, "/"), id))

    response.Header().Add("Location", relativeLocation)

    writeAnswer(response, http.StatusCreated, nil)
}

func (resourceHandler *resourceHandlerAdapter) handleUpdate(request *Request, response http.ResponseWriter) {
    item, err := resourceHandler.readItem(request, Update)
    if err != nil {
        writeError(response, err)
        return
    }

    if err := resourceHandler.resourceHandler.Update(request, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (resourceHandler *resourceHandlerAdapter) handleReplace(request *Request, response http.ResponseWriter) {
    item, err := resourceHandler.readItem(request, Replace)
    if err != nil {
        writeError(response, err)
        return
    }

    if err := resourceHandler.resourceHandler.Replace(request, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (resourceHandler *resourceHandlerAdapter) handleDelete(request *Request, response http.ResponseWriter) {
    if err := resourceHandler.resourceHandler.Delete(request); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (resourceHandler *resourceHandlerAdapter) handleBatchDelete(request *Request, response http.ResponseWriter) {
    if err := resourceHandler.resourceHandler.BatchDelete(request); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (resourceHandler *resourceHandlerAdapter) handleCustomAction(request *Request, handler ActionHandler, response http.ResponseWriter) {
    if err := handler.Do(request); err != nil {
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

func (r *Request) GetParam(key string) string {
    return r.URL.Query().Get(key)
}

func (r *Request) Marshal(v interface{}) ([]byte, error) {
    _, marshal := r.GetMarshalFunc()
    return marshal(v)
}

func (r *Request) GetMarshalFunc() (string, func(interface{}) ([]byte, error)) {
    if r.IsFlagSet("pretty") {
        return "    ", func(v interface{}) ([]byte, error) {
            return json.MarshalIndent(v, "", "    ")
        }
    }

    if indentParam := r.URL.Query().Get("indent"); indentParam != "" {
        if indentCount, err := strconv.Atoi(indentParam); err == nil && 0 < indentCount && indentCount <= 32 {
            indent := strings.Repeat(" ", indentCount)
            return indent, func(v interface{}) ([]byte, error) {
                return json.MarshalIndent(v, "", indent)
            }
        }
    }

    return "", func(v interface{}) ([]byte, error) {
        return json.Marshal(v)
    }
}

func (resourceHandler *resourceHandlerAdapter) readItem(request *Request, action ItemAction) (interface{}, error) {
    itemJson, err := ioutil.ReadAll(request.Body)
    if err != nil {
        return nil, err
    }

    item := resourceHandler.resourceHandler.EmptyItem()

    if err := json.Unmarshal(itemJson, item); err != nil {
        return nil, rest_error.New(http.StatusBadRequest, err.Error())
    }

    if err := CheckRestrictions(item, action); err != nil {
        return nil, rest_error.New(http.StatusBadRequest, err.Error())
    }

    return item, nil
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
