package subsurface_stream

import (
	"context"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
)

type ServerConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
	ReadWriteConfig RWCloserConfig `json:"parser"`
	ClientConfig ClientConfig `json:"client"`
	StreamConfig StreamConfig `json:"stream"`
}

type Server struct {
	*ServerConfig
	ctx context.Context
	cancel context.CancelFunc
	Listener net.Listener
	Client *Client
}

func (config *ServerConfig) getNetwork() string {
	if strings.ToLower(config.Network) == "udp" {
		return "udp"
	}
	return "tcp"
}

func (config *ServerConfig) New() (*Server, error) {
	listener , err := net.Listen(config.getNetwork(), config.Address)
	if err != nil {
		return nil, err
	}
	client, err := config.ClientConfig.New()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config,
		ctx,
		cancel,
		listener,
		client,
	}, nil
}

func (s *Server) Run() {
	for {
		select {
		case <- s.ctx.Done():
			break
		default:
			conn, err := s.Listener.Accept()
			if err != nil {
				logrus.WithError(err).Debug("fail to accept connect")
				continue
			}
			rw1, err := s.ReadWriteConfig.New(conn)
			if err != nil {
				logrus.WithError(err).Error("fail create read writer")
				continue
			}
			var rw2 io.ReadWriteCloser
			if s.Client == nil {
				rw2, err = rw1.DecodeRWCloser()
				if err != nil {
					logrus.WithError(err).Debug("fail decode rw closer")
					rw1.Close()
					continue
				}
			} else {
				rw2 = s.Client.GetRWCloser()
			}
			stream, err := s.StreamConfig.New(rw1, rw2)
			if err != nil {
				logrus.WithError(err).Error("fail to create stream")
				rw1.Close()
				rw2.Close()
			}
			err = stream.Run()
			if err != nil {
				logrus.WithError(err).Error("fail to run stream")
				stream.Close()
			}
		}
	}
}

func (s *Server) Close() error {
	s.cancel()
	return nil
}