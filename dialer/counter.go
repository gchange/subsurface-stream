package dialer

import (
	"errors"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"time"
)

type CounterConfig struct {
	Interval string `subsurface:"interval"`
	Dialer map[string]interface{} `subsurface:"dialer"`
	interval time.Duration
	dialerConfig Config
}

type Counter struct {
	*CounterConfig
	dialer Dialer
	ch chan [2]string
	ticker *time.Ticker
}

func (config *CounterConfig) Init() error {
	intervalLen := len(config.Interval)
	if config.Interval == "" || intervalLen <= 1 {
		return errors.New("invalid interval")
	}
	var interval string
	var duration time.Duration
	switch {
	case config.Interval[intervalLen-1] == 's' || config.Interval[intervalLen-1] == 'S':
		interval = config.Interval[:intervalLen-1]
		duration = time.Second
	case config.Interval[intervalLen-1] == 'm' || config.Interval[intervalLen-1] == 'M':
		interval = config.Interval[:intervalLen-1]
		duration = time.Minute
	case config.Interval[intervalLen-1] == 'h' || config.Interval[intervalLen-1] == 'H':
		interval = config.Interval[:intervalLen-1]
		duration = time.Hour
	default:
		interval = config.Interval
		duration = time.Millisecond
	}
	t, err := strconv.Atoi(interval)
	if err != nil {
		return err
	}
	config.interval = time.Duration(t)*duration
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

func (config *CounterConfig) Clone() Config {
	return &CounterConfig{
		Interval:config.Interval,
		Dialer:config.Dialer,
		interval: config.interval,
		dialerConfig:config.dialerConfig,
	}
}

func (config *CounterConfig) New() (Dialer, error) {
	dialer, err := config.dialerConfig.New()
	if err != nil {
		return nil, err
	}
	counter := &Counter{
		config,
		dialer,
		make(chan [2]string, 512),
		time.NewTicker(config.interval),
	}
	go counter.count()
	return counter, nil
}

func (counter *Counter) Dial(network, address string) (net.Conn, error) {
	select {
	case counter.ch<-[2]string{network, address}:
	default:
		fields := logrus.Fields{
			"network": network,
			"address": address,
		}
		logrus.WithFields(fields).Info("add counter failed")
	}
	return counter.dialer.Dial(network, address)
}

func (counter *Counter) count() {
	fields := logrus.Fields{}
	for {
		select {
		case val:=<-counter.ch:
			d, ok := fields[val[0]]
			if !ok {
				d = make(map[string]int, 0)
				fields[val[0]] = d
			}
			if d, ok := d.(map[string]int); ok {
				if c, ok := d[val[1]];ok {
					d[val[1]] = c+1
				} else {
					d[val[1]] = 1
				}
			} else {
				logrus.WithField("key", val[0]).Error("wrong counter type")
				delete(fields, val[0])
			}
			case <-counter.ticker.C:
				if len(fields) != 0 {
					logrus.WithFields(fields).Info("counter")
					fields = logrus.Fields{}
				}
		}
	}
}

func init() {
	config := &CounterConfig{}
	Register("counter", config)
}
