package rest_client

import (
    "encoding/json"
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io/ioutil"
    "net/http"
    "net/url"
    "strings"
)

type Doable interface {
    Do(method, contentType string, content []byte) (*http.Response, error)
}

type CollectionItem interface {
    Doable
}

type Collection interface {
    Doable

    SubCollection(parentItemId, name string) Collection
    With(paramName string, paramValues... string) Collection
    Ignoring(errorCodes... int) Collection

    Item(itemId string) CollectionItem

    List(items interface{}) error
    ListJson() ([]byte, error)
    ListYaml() ([]byte, error)

    Get(id string, item interface{}) error
    GetJson(id string) ([]byte, error)
    GetYaml(id string) ([]byte, error)

    Create(item interface{}) (string, error)
    CreateJson(itemJson []byte) (string, error)
    CreateYaml(itemYaml []byte) (string, error)

    Update(id string, item interface{}) error
    UpdateJson(id string, itemJson []byte) error
    UpdateYaml(id string, itemYaml []byte) error

    Replace(id string, item interface{}) error
    ReplaceJson(id string, itemJson []byte) error
    ReplaceYaml(id string, itemYaml []byte) error

    Delete(id string) error
}

type _collection struct {
    path        string
    query       url.Values
    ignoreCodes map[int]bool
    client      *Client
}

func (client *Client) Collection(name string) Collection {
    return &_collection{
        path: strings.Trim(name, "/"),
        client: client,
    }
}

func (collection *_collection) SubCollection(parentItemId, name string) Collection {
    return &_collection{
        path: fmt.Sprintf("%s/%s/%s", collection.path, strings.Trim(parentItemId, "/"), strings.Trim(name, "/")),
        query: collection.query,
        ignoreCodes: collection.ignoreCodes,
        client: collection.client,
    }
}

func (collection *_collection) With(paramName string, paramValues... string) Collection {
    newQuery := make(url.Values)
    if collection.query != nil {
        for key, values := range collection.query {
            newQuery[key] = values
        }
    }

    newQuery[paramName] = paramValues

    return &_collection{
        path: collection.path,
        query: newQuery,
        ignoreCodes: collection.ignoreCodes,
        client: collection.client,
    }
}

func (collection *_collection) Ignoring(errorCodes... int) Collection {
    newIgnoreCodes := make(map[int]bool)
    if collection.ignoreCodes != nil {
        for key, value := range collection.ignoreCodes {
            newIgnoreCodes[key] = value
        }
    }

    for _, errorCode := range errorCodes {
        newIgnoreCodes[errorCode] = true
    }

    return &_collection{
        path: collection.path,
        query: collection.query,
        ignoreCodes: newIgnoreCodes,
        client: collection.client,
    }
}

func (collection *_collection) Item(itemId string) CollectionItem {
    return &_collection{
        path: collection.itemPath(itemId),
        query: collection.query,
        ignoreCodes: collection.ignoreCodes,
        client: collection.client,
    }
}

func (collection *_collection) List(items interface{}) error {
    itemsJson, err := collection.ListJson()
    if err != nil || itemsJson == nil {
        return err
    }
    return json.Unmarshal(itemsJson, items)
}

func (collection *_collection) ListJson() ([]byte, error) {
    return collection.doGet(collection.path, Json)
}

func (collection *_collection) ListYaml() ([]byte, error) {
    return collection.doGet(collection.path, Yaml)
}

func (collection *_collection) Get(id string, item interface{}) error {
    itemJson, err := collection.GetJson(id)
    if err != nil || itemJson == nil {
        return err
    }
    return json.Unmarshal(itemJson, item)
}

func (collection *_collection) GetJson(id string) ([]byte, error) {
    return collection.doGet(collection.itemPath(id), Json)
}

func (collection *_collection) GetYaml(id string) ([]byte, error) {
    return collection.doGet(collection.itemPath(id), Yaml)
}

func (collection *_collection) doGet(path, contentType string) ([]byte, error) {
    response, err := collection.do("GET", path, contentType, nil)
    if err != nil || response == nil {
        return nil, err
    }
    return ioutil.ReadAll(response.Body)
}

func (collection *_collection) Create(item interface{}) (string, error) {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return "", err
    }

    response, id, err := collection.doCreate(Json, itemJson)
    if response != nil && err == http.ErrNoLocation {
        if newItemJson, err2 := ioutil.ReadAll(response.Body); err2 == nil {
            if json.Unmarshal(newItemJson, item) == nil {
                err = nil
            }
        }
    }

    return id, err
}

func (collection *_collection) CreateJson(itemJson []byte) (string, error) {
    _, id, err := collection.doCreate(Json, itemJson)
    return id, err
}

func (collection *_collection) CreateYaml(itemYaml []byte) (string, error) {
    _, id, err := collection.doCreate(Yaml, itemYaml)
    return id, err
}

func (collection *_collection) doCreate(contentType string, itemContent []byte) (*http.Response, string, error) {
    response, err := collection.do("POST", collection.path, contentType, itemContent)
    if err != nil || response == nil {
        return nil, "", err
    }

    location, err := response.Location()
    if err != nil {
        return response, "", err
    }

    return response, strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(location.Path, "/"), collection.path), "/"), nil
}

func (collection *_collection) Update(id string, item interface{}) error {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return err
    }
    return collection.UpdateJson(id, itemJson)
}

func (collection *_collection) UpdateJson(id string, itemJson []byte) error {
    return collection.doUpdate(id, Json, itemJson)
}

func (collection *_collection) UpdateYaml(id string, itemYaml []byte) error {
    return collection.doUpdate(id, Yaml, itemYaml)
}

func (collection *_collection) doUpdate(id, contentType string, itemContent []byte) error {
    _, err := collection.do("POST", collection.itemPath(id), contentType, itemContent)
    return err
}

func (collection *_collection) Replace(id string, item interface{}) error {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return err
    }
    return collection.ReplaceJson(id, itemJson)
}

func (collection *_collection) ReplaceJson(id string, itemJson []byte) error {
    return collection.doReplace(id, Json, itemJson)
}

func (collection *_collection) ReplaceYaml(id string, itemYaml []byte) error {
    return collection.doReplace(id, Yaml, itemYaml)
}

func (collection *_collection) doReplace(id, contentType string, itemContent []byte) error {
    _, err := collection.do("PUT", collection.itemPath(id), contentType, itemContent)
    return err
}

func (collection *_collection) Delete(id string) error {
    _, err := collection.do("DELETE", collection.itemPath(id), "", nil)
    return err
}

func (collection *_collection) Do(method, contentType string, content []byte) (*http.Response, error) {
    return collection.do(method, collection.path, contentType, content)
}

func (collection *_collection) do(method, path, contentType string, content []byte) (*http.Response, error) {
    queryPath := path

    if collection.query != nil && len(collection.query) != 0 {
        queryPath = fmt.Sprintf("%s?%s", path, collection.query.Encode())
    }

    response, err := collection.client.Do(method, queryPath, contentType, content)

    if err != nil && collection.ignoreCodes != nil {
        if restErr, ok := err.(*rest_error.Error); ok && collection.ignoreCodes[restErr.Code] {
            return nil, nil
        }
    }

    return response, err
}

func (collection *_collection) itemPath(id string) string {
    return fmt.Sprintf("%s/%s", collection.path, strings.Trim(id, "/"))
}
