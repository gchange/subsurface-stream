package stream

import (
	"github.com/gchange/subsurface-stream/dialer"
	"github.com/gchange/subsurface-stream/socks5"
	"net"
)

type Socks5Config struct {
	Network string `subsurface:"network"`
	Address string `subsurface:"address"`
	Dialer map[string]interface{} `subsurface:"dialer"`
	dialer dialer.Dialer
}

func (config *Socks5Config) Init() error {
	dialerConfig, err := dialer.GetDialerConfig(config.Dialer)
	if err != nil {
		return err
	}
	err = dialerConfig.Init()
	if err != nil {
		return err
	}
	config.dialer, err = dialerConfig.New()
	if err != nil {
		return err
	}
	return nil
}

func (config *Socks5Config) Clone() Config {
	return &Socks5Config{
		Network: config.Network,
		Address:config.Address,
		Dialer: config.Dialer,
		dialer:config.dialer,
	}
}

func (config *Socks5Config) New(conn net.Conn) (net.Conn, error) {
	if config.Address == "" {
		return socks5.Socks5Server(conn, config.dialer)
	}
	return socks5.Socks5Proxy(conn, config.dialer, config.Network, config.Address)
}

func init() {
	config := &Socks5Config{
		Network: "tcp",
	}
	Register("socks5", config)
}
