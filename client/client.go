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

const (
    Json = "application/json"
    Yaml = "application/yaml"
)

type Client struct {
    serverUrl  string
    httpClient *http.Client
}

func New(serverUrl string, httpClient *http.Client) *Client {
    return &Client{
        serverUrl: strings.TrimSuffix(serverUrl, "/"),
        httpClient: httpClient,
    }
}

func (client *Client) WithPrefix(prefix string) *Client {
    return New(client.path(prefix), client.httpClient)
}

type Header struct {
    Name   string
    Values []string
}

func (client *Client) Do(method, path, contentType string, content []byte, additionalHeaders... *Header) (*http.Response, error) {
    var contentReader io.Reader
    if content != nil {
        contentReader = bytes.NewReader(content)
    }
    return client.DoStream(method, path, contentType, contentReader, additionalHeaders...)
}

func (client *Client) DoStream(method, path, contentType string, contentReader io.Reader, additionalHeaders... *Header) (*http.Response, error) {
    request, err := http.NewRequest(method, client.path(path), contentReader)
    if err != nil {
        return nil, err
    }

    if contentType != "" {
        if contentReader != nil {
            request.Header.Add("Content-Type", contentType)
        }
        request.Header.Add("Accept", contentType)
    }

    request.Header.Add("User-Agent", "curl/7.43.0")

    for _, header := range additionalHeaders {
        if header.Name != "" && header.Values != nil {
            request.Header[header.Name] = header.Values
        }
    }

    response, err := client.httpClient.Do(request)
    if err != nil {
        return nil, err
    }

    if response.StatusCode / 100 == 2 {
        return response, nil
    }
    defer response.Body.Close()

    message, err := ioutil.ReadAll(response.Body)

    if err == nil && message != nil {
        return nil, rest_error.New(response.StatusCode, fmt.Sprintf("\n%s", string(message)))
    } else {
        return nil, rest_error.NewByCode(response.StatusCode)
    }
}

func (client *Client) path(path string) string {
    return fmt.Sprintf("%s/%s", client.serverUrl, strings.TrimPrefix(path, "/"))
}
