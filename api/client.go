package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/itsHenry35/tal_downloader/config"
	"github.com/itsHenry35/tal_downloader/constants"
)

type Client struct {
	httpClient *http.Client
	token      string
	userID     string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetAuth(token, userID string) {
	if token != "" {
		c.token = token
	}
	if userID != "" {
		c.userID = userID
	}
}

// returns token, userID
func (c *Client) GetAuth() (string, string) {
	return c.token, c.userID
}

func (c *Client) getDefaultHeaders() map[string]string {
	return map[string]string{
		"User-Agent": config.UserAgent,
		"Referer":    "https://speiyou.cn/",
		"terminal":   config.Terminal,
		"version":    config.Version,
		"resVer":     config.ResVer,
	}
}

// nodh -> no default headers
func (c *Client) doRequest(method, urlStr string, body interface{}, headers map[string]string, nodh bool) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		switch v := body.(type) {
		case url.Values:
			reqBody = strings.NewReader(v.Encode())
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			reqBody = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return nil, err
	}

	if !nodh {
		// Set default headers
		for k, v := range c.getDefaultHeaders() {
			req.Header.Set(k, v)
		}
	}

	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Add auth headers if available
	if c.token != "" {
		req.Header.Set("token", c.token)
	}
	if c.userID != "" {
		req.Header.Set("stuId", c.userID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if constants.Version == "Debug" {
		fmt.Println("Request URL:", urlStr)
		fmt.Println("Request Method:", method)
		// print request headers
		fmt.Println("Request Headers:", req.Header)
	}
	// print response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Reset the response body so it can be read again later
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	if constants.Version == "Debug" {
		fmt.Println("Response Status:", resp.Status)
		fmt.Println("Response Body:", string(bodyBytes))
	}

	return resp, nil
}
