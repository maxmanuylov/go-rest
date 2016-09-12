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
    url := fmt.Sprintf("%s/%s", client.serverUrl, strings.TrimPrefix(path, "/"))

    request, err := http.NewRequest(method, url, contentReader)
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
            for _, value := range header.Values {
                request.Header.Add(header.Name, value)
            }
        }
    }

    response, err := client.httpClient.Do(request)
    if err != nil {
        return nil, err
    }

    if response.StatusCode / 100 == 2 {
        return response, nil
    }

    message, err := ioutil.ReadAll(response.Body)

    if err == nil && message != nil {
        return nil, rest_error.New(response.StatusCode, fmt.Sprintf("\n%s", string(message)))
    } else {
        return nil, rest_error.NewByCode(response.StatusCode)
    }
}
