package subsurface_stream

import "io"

type ParserConfig interface {
	New() (Parser, error)
}

type Parser interface{
	DecodeRWCloser() (io.ReadWriteCloser, error)
	Decode([]byte, int) (int, error)
	Encode([]byte) ([]byte, error)
}
