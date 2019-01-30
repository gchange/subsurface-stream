package stream

import (
	"errors"
	"github.com/gchange/subsurface-stream/parser"
	"net"
	"reflect"
	"sync"
)

var (
	lock  = sync.RWMutex{}
	pool = map[string]Config{}
)

type Config interface {
	Init() error
	Clone() Config
	New(conn net.Conn) (net.Conn, error)
}

func GetStreamConfig(config map[string]interface{}) (Config, error) {
	var name string
	if n, ok := config["name"]; !ok {
		return nil, errors.New("config name not found")
	} else if name, ok = n.(string); !ok {
		return nil, errors.New("config name type error")
	}

	lock.RLock()
	defer lock.RUnlock()
	if c, ok := pool[name]; ok {
		nc := c.Clone()
		err := parser.Unmarshal("subsurface", reflect.ValueOf(nc), config)
		if err != nil {
			return nil, err
		}
		return nc, nil
	}
	return nil, errors.New("config not found")
}

func Register(name string, config Config) error {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := pool[name]; ok {
		return errors.New("config already exists")
	}
	pool[name] = config
	return nil
}
