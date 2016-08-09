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

    Get(id string, item interface{}) error
    GetJson(id string) ([]byte, error)

    Create(item interface{}) (string, error)
    CreateJson(itemJson []byte) (string, error)

    Update(id string, item interface{}) error
    UpdateJson(id string, itemJson []byte) error

    Replace(id string, item interface{}) error
    ReplaceJson(id string, itemJson []byte) error

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
    return collection.doGet(collection.path)
}

func (collection *collectionClient) Get(id string, item interface{}) error {
    itemJson, err := collection.GetJson(id)
    if err != nil {
        return err
    }
    return json.Unmarshal(itemJson, item)
}

func (collection *collectionClient) GetJson(id string) ([]byte, error) {
    return collection.doGet(collection.itemPath(id))
}

func (collection *collectionClient) doGet(path string) ([]byte, error) {
    response, err := collection.client.Do("GET", path, nil)
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
    response, err := collection.client.Do("POST", collection.path, itemJson)
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
    _, err := collection.client.Do("POST", collection.itemPath(id), itemJson)
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
    _, err := collection.client.Do("PUT", collection.itemPath(id), itemJson)
    return err
}

func (collection *collectionClient) Delete(id string) error {
    _, err := collection.client.Do("DELETE", collection.itemPath(id), nil)
    return err
}

func (collection *collectionClient) itemPath(id string) string {
    return fmt.Sprintf("%s%s", collection.path, id)
}
