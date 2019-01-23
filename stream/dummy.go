package stream

import "io"

type DummyConfig struct {
}

func (config *DummyConfig) Clone() Config {
	return &DummyConfig{}
}

func (config *DummyConfig) New(closer io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	return closer, nil
}

func init() {
	Register("dummy", &DummyConfig{})
}
