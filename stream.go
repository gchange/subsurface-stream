package subsurface_stream

import (
	"github.com/sirupsen/logrus"
	"io"
)

type StreamConfig struct {
}

type Stream struct {
	*StreamConfig
	ReadWriters []io.ReadWriteCloser
}

func (config *StreamConfig) New(rws ...io.ReadWriteCloser) (*Stream, error) {
	return &Stream{
		config,
		rws,
	}, nil
}

func (Stream *Stream) transport(reader io.Reader, writer io.WriteCloser) {
	defer writer.Close()
	for {
		_, err := io.Copy(writer, reader)
		if err != nil {
			logrus.WithError(err).Debug("fail to copy buffer from reader to writer")
			break
		}
	}
}

func (Stream *Stream) Run() (err error) {
	for i:=0; i<len(Stream.ReadWriters)-1;i++ {
		go Stream.transport(Stream.ReadWriters[i], Stream.ReadWriters[i+1])
		go Stream.transport(Stream.ReadWriters[i+1], Stream.ReadWriters[i])
	}
	return nil
}

func (Stream *Stream) Close() (err error) {
	for _, rw := range Stream.ReadWriters {
		if e := rw.Close(); e!=nil {
			logrus.WithError(err).Debug("fail to close read writer")
			err = e
		}
 	}
	return
}
