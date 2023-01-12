package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
)

type Client struct {
	client *http.Client
}

func NewClient(socketAddr string) *Client {

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketAddr)
			},
		},
	}
	return &Client{
		client: c,
	}
}

var ErrNotReady = errors.New("not ready")

func (c *Client) Ready() error {
	res, err := c.client.Get("http://localhost" + "/ready")
	if err != nil {
		return ErrNotReady
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return nil
	}
	return ErrNotReady
}

func (c *Client) Get(path string) (string, error) {
	res, err := c.client.Get("http://localhost" + path)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *Client) GetJson(path string, data any) error {
	res, err := c.client.Get("http://localhost" + path)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Stop() error {
	req, err := http.NewRequest(http.MethodDelete, "http://localhost/stop", nil)
	if err != nil {
		return err
	}
	_, err = c.client.Do(req)
	if err != nil {
		return err
	}
	return nil
}
