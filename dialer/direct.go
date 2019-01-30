package dialer

import "net"

type DirectConfig struct {
}

func (config *DirectConfig) Init() error {
	return nil
}

func (config *DirectConfig) Clone() Config {
	return &DirectConfig{}
}

func (config *DirectConfig) New() (Dialer, error) {
	return &net.Dialer{}, nil
}

func init() {
	Register("direct", &DirectConfig{})
}
