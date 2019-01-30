package stream

import "net"

type TCPConfig struct {
}

func (config *TCPConfig) Init() error {
	return nil
}

func (config *TCPConfig) Clone() Config {
	return &TCPConfig{}
}

func (config *TCPConfig) New(conn net.Conn) (net.Conn, error) {
	return nil, nil
}

func init() {
	config := &TCPConfig{}
	Register("tcp", config)
}
