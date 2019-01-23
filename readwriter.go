package subsurface_stream

import (
	"errors"
	"io"
)

type RWCloserConfig struct {
	Reader []ParserConfig `json:"reader"`
	Writer []ParserConfig `json:"writer"`
}

type RWCloser struct {
	*RWCloserConfig
	rw io.ReadWriteCloser
	rps []Parser
	wps []Parser
}

func (config *RWCloserConfig) New(rw io.ReadWriteCloser) (*RWCloser, error) {
	rps := make([]Parser, len(config.Reader))
	for _, c := range config.Reader {
		parser, err := c.New()
		if err != nil {
			return nil, err
		}
		rps = append(rps, parser)
	}
	wps := make([]Parser, len(config.Writer))
	for _, c := range config.Writer {
		parser, err := c.New()
		if err != nil {
			return nil, err
		}
		wps = append(wps, parser)
	}
	return &RWCloser{config, rw, rps, wps}, nil
}

func (rw *RWCloser) DecodeRWCloser() (io.ReadWriteCloser, error) {
	buf := make([]byte, 1024)
	var err error
	var n int
	var r io.ReadWriteCloser
	for {
		for _, parser := range rw.rps {
			r, err = parser.DecodeRWCloser()
			if err != nil {
				return nil, err
			}
			if rw != nil {
				return r, nil
			}
		}
		n, err = rw.Read(buf)
		if err != nil {
			return nil, err
		}
		if n != 0 {
			return nil, errors.New("return data before decode rw closer")
		}
	}
}

func (rw *RWCloser) read(buf []byte, index, n int) (int, error) {
	if index < 0 {
		return rw.rw.Read(buf)
	}
	if index >= len(rw.rps) {
		return 0, errors.New("parser index error")
	}
	var err error
	for {
		n, err = rw.rps[index].Decode(buf, n)
		if err != nil {
			return 0, err
		}
		if n != 0 {
			return n, nil
		}
		n, err = rw.read(buf, index-1, n)
	}
}

func (rw *RWCloser) Read(buf []byte) (int, error) {
	return rw.read(buf, len(rw.rps)-1, 0)
}

func (rw *RWCloser) Write(buf []byte) (int, error) {
	var err error
	for _, parser := range rw.wps {
		if buf, err = parser.Encode(buf); err != nil {
			return 0, err
		} else if len(buf) == 0 {
			return 0, nil
		}
	}
	return rw.rw.Write(buf)
}

func (rw *RWCloser) Close() error {
	buf := make([]byte, 0)
	for {
		n, err := rw.Write(buf)
		if err != nil || n == 0 {
			break
		}
	}
	return rw.rw.Close()
}
