package dialer

import (
	"errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type PoolConfig struct {
	IdleTime uint `subsurface:"idle_time"`
	MaxConnect uint `subsurface:"max_connect"`
	Dialer map[string]interface{} `subsurface:"dialer"`
	dialerConfig Config
}

type conn struct {
	net.Conn
	ch chan *conn
}

type pool struct {
	network string
	address string
	ch chan *conn
	flag int32
	ticker *time.Ticker
}

type Pool struct {
	*PoolConfig
	dialer Dialer
	pool map[[2]string]*pool
	lock sync.RWMutex
}

func (config *PoolConfig) Init() error {
	var err error
	config.dialerConfig, err = GetDialerConfig(config.Dialer)
	if err != nil {
		return err
	}
	err = config.dialerConfig.Init()
	if err != nil {
		return err
	}
	return nil
}

func (config *PoolConfig) Clone() Config {
	return &PoolConfig{
		IdleTime:config.IdleTime,
		MaxConnect:config.MaxConnect,
		Dialer:config.Dialer,
		dialerConfig:config.dialerConfig,
	}
}

func (config *PoolConfig) New() (Dialer, error) {
	dialer, err := config.dialerConfig.New()
	if err != nil {
		return nil, err
	}
	return &Pool{
		config,
		dialer,
		make(map[[2]string]*pool, 0),
		sync.RWMutex{},
	}, nil
}

func (c *conn) Close() error {
	select {
	case c.ch <- c:
		return nil
	default:
		logrus.Error("push connect back to poll failed")
		return c.Close()
	}
}

func (p *pool) get() net.Conn {
	atomic.CompareAndSwapInt32(&p.flag, 0, 1)
	select {
	case conn := <- p.ch:
		return conn
	default:
		return nil
	}

}

func (p *pool) new(dialer Dialer, network, address string) (net.Conn, error) {
	if network != p.network || address != p.address {
		return nil, errors.New("can not create difference connection in a pool")
	}
	c := p.get()
	if c != nil {
		return c, nil
	}
	c, err := dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	c2 := &conn{c,p.ch}
	select {
	case p.ch<- c2:
		return c2, nil
	default:
		return c, nil
	}
}

func (p *pool) release() {
	if !atomic.CompareAndSwapInt32(&p.flag, 1, 0) {
		for conn := range p.ch {
			if err := conn.Conn.Close(); err != nil {
				logrus.WithError(err).Debug("close connection in pool failed")
			}
		}
	}
}

func (p *pool) close() {
	p.ticker.Stop()
	close(p.ch)
	for conn := range p.ch {
		if err := conn.Conn.Close(); err != nil {
			logrus.WithError(err).Debug("close connection in pool failed")
		}
	}
}

func (p *Pool) Dial(network, address string) (net.Conn, error) {
	sp, err := p.get(network, address)
	if err != nil {
		return nil, err
	}
	if sp == nil {
		sp, err = p.create(network, address)
		if err != nil {
			return nil, err
		}
	}
	if sp == nil {
		return nil, errors.New("create connection poll failed")
	}
	conn, err := sp.new(p.dialer, network, address)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (p *Pool) get(network, address string) (*pool, error) {
	key := [2]string{network, address}
	p.lock.RLock()
	defer p.lock.RUnlock()
	if sp, ok := p.pool[key]; ok {
		return sp, nil
	}
	return nil, nil
}

func (p *Pool) create(network, address string) (*pool, error) {
	key := [2]string{network, address}
	p.lock.Lock()
	defer p.lock.Unlock()
	if sp, ok := p.pool[key]; ok {
		return sp, nil
	}
	ch := make(chan *conn, p.MaxConnect)
	ticker := time.NewTicker(time.Duration(p.IdleTime)*time.Second)
	sp := &pool{
		network:network,
		address:address,
		ch: ch,
		ticker:ticker,
		flag:0,
	}
	go func() {
		for range sp.ticker.C {
			sp.release()
		}
	}()
	p.pool[key] = sp
	return sp, nil
}
