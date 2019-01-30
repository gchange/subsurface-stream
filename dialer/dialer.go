package dialer

import (
	"github.com/gchange/subsurface-stream/parser"
	"github.com/pkg/errors"
	"net"
	"reflect"
	"sync"
)

var (
	lock = sync.RWMutex{}
	dialerPool = map[string]Config{}
)

type Config interface {
	Init() error
	Clone() Config
	New() (Dialer, error)
}

type Dialer interface {
	Dial(string, string) (net.Conn, error)
}

func Register(name string, config Config) error {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := dialerPool[name]; ok {
		return errors.New("config already exists")
	} else {
		dialerPool[name] = config
	}
	return nil
}

func GetDialerConfig(config map[string]interface{}) (Config, error) {
	var name string
	if n, ok := config["name"]; !ok {
		return nil, errors.New("config name not found")
	} else if name, ok = n.(string); !ok {
		return nil, errors.New("config name type error")
	}

	lock.RLock()
	defer lock.RUnlock()
	if c, ok := dialerPool[name]; ok {
		nc := c.Clone()
		err := parser.Unmarshal("subsurface", reflect.ValueOf(nc), config)
		if err != nil {
			return nil, err
		}
		return nc, nil
	}
	return nil, errors.New("config not found")
}
