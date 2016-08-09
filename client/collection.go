package rest_client

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "strings"
)

type CollectionClient interface {
    SubCollection(parentItemId, name string) CollectionClient

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

type collectionClient struct {
    path    string
    client  *Client
}

func (client *Client) Collection(name string) CollectionClient {
    return &collectionClient{
        path: fmt.Sprintf("%s/", strings.Trim(name, "/")),
        client: client,
    }
}

func (collection *collectionClient) SubCollection(parentItemId, name string) CollectionClient {
    return &collectionClient{
        path: fmt.Sprintf("%s%s/%s/", collection.path, strings.Trim(parentItemId, "/"), strings.Trim(name, "/")),
        client: collection.client,
    }
}

func (collection *collectionClient) List(items interface{}) error {
    itemsJson, err := collection.ListJson()
    if err != nil {
        return err
    }
    return json.Unmarshal(itemsJson, items)
}

func (collection *collectionClient) ListJson() ([]byte, error) {
    return collection.doGet(collection.path, Json)
}

func (collection *collectionClient) ListYaml() ([]byte, error) {
    return collection.doGet(collection.path, Yaml)
}

func (collection *collectionClient) Get(id string, item interface{}) error {
    itemJson, err := collection.GetJson(id)
    if err != nil {
        return err
    }
    return json.Unmarshal(itemJson, item)
}

func (collection *collectionClient) GetJson(id string) ([]byte, error) {
    return collection.doGet(collection.itemPath(id), Json)
}

func (collection *collectionClient) GetYaml(id string) ([]byte, error) {
    return collection.doGet(collection.itemPath(id), Yaml)
}

func (collection *collectionClient) doGet(path, contentType string) ([]byte, error) {
    response, err := collection.client.Do("GET", path, contentType, nil)
    if err != nil {
        return nil, err
    }
    return ioutil.ReadAll(response.Body)
}

func (collection *collectionClient) Create(item interface{}) (string, error) {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return "", err
    }
    return collection.CreateJson(itemJson)
}

func (collection *collectionClient) CreateJson(itemJson []byte) (string, error) {
    return collection.doCreate(Json, itemJson)
}

func (collection *collectionClient) CreateYaml(itemYaml []byte) (string, error) {
    return collection.doCreate(Yaml, itemYaml)
}

func (collection *collectionClient) doCreate(contentType string, itemContent []byte) (string, error) {
    response, err := collection.client.Do("POST", collection.path, contentType, itemContent)
    if err != nil {
        return "", err
    }

    location, err := response.Location()
    if err != nil {
        return "", err
    }

    return strings.TrimPrefix(strings.TrimPrefix(location.Path, "/"), collection.path), nil
}

func (collection *collectionClient) Update(id string, item interface{}) error {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return err
    }
    return collection.UpdateJson(id, itemJson)
}

func (collection *collectionClient) UpdateJson(id string, itemJson []byte) error {
    return collection.doUpdate(id, Json, itemJson)
}

func (collection *collectionClient) UpdateYaml(id string, itemYaml []byte) error {
    return collection.doUpdate(id, Yaml, itemYaml)
}

func (collection *collectionClient) doUpdate(id, contentType string, itemContent []byte) error {
    _, err := collection.client.Do("POST", collection.itemPath(id), contentType, itemContent)
    return err
}

func (collection *collectionClient) Replace(id string, item interface{}) error {
    itemJson, err := json.Marshal(item)
    if err != nil {
        return err
    }
    return collection.ReplaceJson(id, itemJson)
}

func (collection *collectionClient) ReplaceJson(id string, itemJson []byte) error {
    return collection.doReplace(id, Json, itemJson)
}

func (collection *collectionClient) ReplaceYaml(id string, itemYaml []byte) error {
    return collection.doReplace(id, Yaml, itemYaml)
}

func (collection *collectionClient) doReplace(id, contentType string, itemContent []byte) error {
    _, err := collection.client.Do("PUT", collection.itemPath(id), contentType, itemContent)
    return err
}

func (collection *collectionClient) Delete(id string) error {
    _, err := collection.client.Do("DELETE", collection.itemPath(id), "", nil)
    return err
}

func (collection *collectionClient) itemPath(id string) string {
    return fmt.Sprintf("%s%s", collection.path, id)
}
