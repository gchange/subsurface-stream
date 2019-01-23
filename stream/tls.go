package stream

import (
	"crypto/tls"
	"io"
)

type TLSConfig struct {
}

type TLSStream struct {
}

func (config *TLSConfig) Clone() Config {
	return &TLSConfig{}
}

func (config *TLSConfig) New(closer io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	tls.Client()
	return &TLSStream{}, nil
}

func (tls *TLSStream) Read(buf []byte) (int, error) {
	return len(buf), nil
}

func (tls *TLSStream) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func (tls *TLSStream) Close() error {
	return nil
}

func init() {
	err := Register("tls", &TLSConfig{})
	if err != nil {
		panic(err)
	}
}
