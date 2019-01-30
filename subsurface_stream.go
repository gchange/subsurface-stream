package subsurface_stream

import (
	"github.com/gchange/subsurface-stream/steam"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type Config struct {
	Network string `json:"network"`
	Address string `json:"address"`
	Configs []map[string]interface{} `json:"config"`
}

type SubsurfaceStream struct {
	*Config
	Streams []stream.Config
	listener net.Listener
	pool map[net.Conn]bool
	lock sync.RWMutex
}

func (config *Config) New() (*SubsurfaceStream, error) {
	var err error
	streams := make([]stream.Config, len(config.Configs))
	for i, m := range config.Configs {
		streams[i], err = stream.GetStreamConfig(m)
		if err != nil {
			return nil, err
		}
		err = streams[i].Init()
		if err != nil {
			return nil, err
		}
	}
	listener, err := net.Listen(config.Network, config.Address)
	if err != nil {
		return nil, err
	}
	return &SubsurfaceStream{
		config,
		streams,
		listener,
		make(map[net.Conn]bool, 0),
		sync.RWMutex{},
	}, nil
}

func (ss *SubsurfaceStream) accept(conn net.Conn) {
	var err error
	defer func() {
		if err != nil && conn != nil {
			conn.Close()
		}
	}()

	for _, stream := range ss.Streams {
		conn, err = stream.New(conn)
		if err != nil {
			break
		}
	}
}

func (ss *SubsurfaceStream) Run() {
	for {
		conn, err := ss.listener.Accept()
		if err != nil {
			logrus.WithError(err).Debug("failed to accept connection")
			continue
		}
		go ss.accept(conn)
	}
}

func (ss *SubsurfaceStream) Close() error {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	err := ss.listener.Close()
	if err != nil {
		logrus.WithError(err).Debug("fail to close listener")
	}
	for conn := range ss.pool {
		if e := conn.Close(); e != nil {
			logrus.WithError(e).Debug("fail to close pool connection")
			err = e
		}
	}
	ss.pool = nil
	return err
}
