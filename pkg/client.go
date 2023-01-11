package pkg

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
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

func (c *Client) Get(path string) {
	res, err := c.client.Get("http://localhost" + path)
	if err != nil {
		return
	}
	io.Copy(os.Stdout, res.Body)
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
