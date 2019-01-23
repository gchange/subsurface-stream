package subsurface_stream

import (
	"io"
	"net"
	"strings"
)

type ClientConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
	ReadWriter RWCloserConfig `json:"parser"`
}

type Client struct {
	*ClientConfig
	ReadWriter io.ReadWriteCloser
}

func (config *ClientConfig) getNetwork() string {
	if strings.ToLower(config.Network) == "udp" {
		return "udp"
	}
	return "tcp"
}

func (config *ClientConfig) New() (*Client, error) {
	if config.Address == "" {
		return nil, nil
	}

	var rw *RWCloser
	if config.Address != "" {
		conn, err := net.Dial(config.getNetwork(), config.Address)
		if err != nil {
			return nil, err
		}
		rw, err = config.ReadWriter.New(conn)
		if err != nil {
			return nil, err
		}
	}
	return &Client{config, rw}, nil
}

func (client *Client) GetRWCloser() io.ReadWriteCloser {
	return client.ReadWriter
}

func (client *Client) Close() error {
	return nil
}
