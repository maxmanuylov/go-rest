package rest

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "path"
    "strings"
)

type ResourceHandler interface {
    EmptyItem() interface{}
    Create(ids []string, item interface{}) (string, error)
    Read(ids []string) (interface{}, error)
    List(ids []string) ([]interface{}, error)
    Update(ids []string, item interface{}) error
    Replace(ids []string, item interface{}) error
    Delete(ids []string) error
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

    pattern := fmt.Sprintf("/%s/", name)
    server.mux.HandleFunc(pattern, func(response http.ResponseWriter, request *http.Request) {
        path := strings.TrimPrefix(request.URL.Path, "/")
        path = strings.TrimPrefix(path, fmt.Sprintf("%s", name))
        path = strings.Trim(path, "/")

        pathNames := strings.Split(path, "/")

        ids := make([]string, 0)
        actualCollection := collection
        id := true

        for _, pathName := range pathNames {
            if id {
                ids = append(ids, pathName)
                id = false
            } else {
                actualCollection = actualCollection.subCollections[pathName]
                if actualCollection == nil {
                    writeError(response, ErrNotFound)
                    return
                }
                id = true
            }
        }

        actualCollection.handle(ids, response, request)
    })

    return collection
}

func (collection *Collection) SubCollection(name string, handler ResourceHandler) *Collection {
    subCollection := newCollection(collection.level + 1, handler)
    collection.subCollections[name] = subCollection
    return subCollection
}

func (collection *Collection) handle(ids []string, response http.ResponseWriter, request *http.Request) {
    collectionRequest := collection.level == len(ids)

    switch strings.ToUpper(request.Method) {
    case "GET":
        if collectionRequest {
            collection.handleList(ids, response, request)
            return
        } else {
            collection.handleRead(ids, response, request)
            return
        }

    case "POST":
        if collectionRequest {
            collection.handleCreate(ids, response, request)
            return
        } else {
            collection.handleUpdate(ids, response, request)
            return
        }

    case "PUT":
        if !collectionRequest {
            collection.handleReplace(ids, response, request)
            return
        }

    case "DELETE":
        if !collectionRequest {
            collection.handleDelete(ids, response, request)
            return
        }
    }

    writeError(response, ErrMethodNotAllowed)
}

func (collection *Collection) handleList(ids []string, response http.ResponseWriter, request *http.Request) {
    items, err := collection.handler.List(ids)
    if err != nil {
        writeError(response, err)
        return
    }

    itemsJson, err := json.Marshal(items)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemsJson)
}

func (collection *Collection) handleRead(ids []string, response http.ResponseWriter, request *http.Request) {
    item, err := collection.handler.Read(ids)
    if err != nil {
        writeError(response, err)
        return
    }

    if item == nil {
        writeError(response, ErrNotFound)
        return
    }

    itemJson, err := json.Marshal(item)
    if err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, itemJson)
}

func (collection *Collection) handleCreate(ids []string, response http.ResponseWriter, request *http.Request) {
    item, err := collection.readItem(request, "create")
    if err != nil {
        writeError(response, err)
        return
    }

    id, err := collection.handler.Create(ids, item)
    if err != nil {
        writeError(response, err)
        return
    }

    url := request.URL
    resourcePath := path.Clean(fmt.Sprintf("%s/%s", strings.Trim(url.Path, "/"), id))
    resourceLocation := fmt.Sprintf("%s://%s/%s", url.Scheme, url.Host, resourcePath)

    response.Header().Add("Location", resourceLocation)

    writeAnswer(response, http.StatusCreated, nil)
}

func (collection *Collection) handleUpdate(ids []string, response http.ResponseWriter, request *http.Request) {
    item, err := collection.readItem(request, "update")
    if err != nil {
        writeError(response, err)
        return
    }

    if err := collection.handler.Update(ids, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (collection *Collection) handleReplace(ids []string, response http.ResponseWriter, request *http.Request) {
    item, err := collection.readItem(request, "replace")
    if err != nil {
        writeError(response, err)
        return
    }

    if err := collection.handler.Replace(ids, item); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}

func (collection *Collection) handleDelete(ids []string, response http.ResponseWriter, request *http.Request) {
    if err := collection.handler.Delete(ids); err != nil {
        writeError(response, err)
        return
    }

    writeAnswer(response, http.StatusOK, nil)
}


func (collection *Collection) readItem(request *http.Request, action string) (interface{}, error) {
    itemJson, err := ioutil.ReadAll(request.Body)
    if err != nil {
        return nil, err
    }

    item := collection.handler.EmptyItem()

    if err := json.Unmarshal(itemJson, item); err != nil {
        return nil, &Error{code: http.StatusBadRequest, message: err.Error()}
    }

    if err := checkRequiredFields(item, action); err != nil {
        return nil, &Error{code: http.StatusBadRequest, message: err.Error()}
    }

    return item, nil
}

func writeError(response http.ResponseWriter, err error) {
    code := http.StatusInternalServerError
    if restError, ok := err.(*Error); ok {
        code = restError.code
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