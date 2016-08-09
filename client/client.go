package rest_client

import (
    "bytes"
    "fmt"
    "github.com/maxmanuylov/go-rest/error"
    "io"
    "io/ioutil"
    "net/http"
    "strings"
)

type Client struct {
    serverUrl  string
    httpClient *http.Client
}

func New(serverUrl string, httpClient *http.Client) *Client {
    return &Client{
        serverUrl: serverUrl,
        httpClient: httpClient,
    }
}

func (client *Client) Do(method, path string, contentJson []byte) (*http.Response, error) {
    var contentReader io.Reader
    if contentJson != nil {
        contentReader = bytes.NewReader(contentJson)
    }

    url := fmt.Sprintf("%s/%s", strings.TrimSuffix(client.serverUrl, "/"), strings.TrimPrefix(path, "/"))

    request, err := http.NewRequest(method, url, contentReader)
    if err != nil {
        return nil, err
    }

    request.Header.Add("Content-Type", "application/json")
    request.Header.Add("Accept", "application/json")
    request.Header.Add("User-Agent", "curl/7.43.0")

    response, err := client.httpClient.Do(request)
    if err != nil {
        return nil, err
    }

    if response.StatusCode / 100 == 2 {
        return response, nil
    }

    message, err := ioutil.ReadAll(response.Body)

    if err == nil && message != nil {
        return nil, rest_error.New(response.StatusCode, string(message))
    } else {
        return nil, rest_error.NewByCode(response.StatusCode)
    }
}
