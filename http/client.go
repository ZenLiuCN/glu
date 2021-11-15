package http

import (
	"net/http"
	"strings"
	"time"
)

type Client struct {
	http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{Client: http.Client{
		Timeout: timeout,
	}}
}

func (c *Client) Get(url string) (*http.Response, error) {
	return c.Client.Get(url)
}
func (c *Client) Post(url, contentType, data string) (*http.Response, error) {
	return c.Client.Post(url, contentType, strings.NewReader(data))
}
func (c *Client) Head(url string) (*http.Response, error) {
	return c.Client.Head(url)
}
func (c *Client) Form(url string, frm map[string][]string) (*http.Response, error) {
	return c.Client.PostForm(url, frm)
}
func (c *Client) Do(method, url, data string, header map[string]string) (res *http.Response, err error) {
	var req *http.Request
	if data == "" {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequest(method, url, strings.NewReader(data))
		if err != nil {
			return nil, err
		}
	}
	if header != nil {
		for s, s2 := range header {
			req.Header.Set(s, s2)
		}
	}
	return c.Client.Do(req)
}
