package subsurface_stream

import "io"

type DummyParserConfig struct {
}

type DummyParser struct {
	*DummyParserConfig
}

func (config *DummyParserConfig) New() (Parser, error) {
	return &DummyParser{config}, nil
}

func (parser *DummyParser) DecodeRWCloser() (io.ReadWriteCloser, error) {
	return nil, nil
}

func (parser *DummyParser) Decode(buf []byte, n int) (int, error) {
	return n, nil
}

func (parser *DummyParser) Encode(buf []byte) ([]byte, error) {
	return buf, nil
}
