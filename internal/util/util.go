package util

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// DownloadFile downloads the file at the given URL using the HTTP GET method and returns its contents and MIME type.
func DownloadFile(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("request failed: %v", resp.Status)
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(contents)
	}
	return contents, contentType, nil
}
