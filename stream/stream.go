package stream

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var (
	streams = make(map[string]Config)
	lock = sync.RWMutex{}
)

type Config interface {
	Clone() Config
	New(closer io.ReadWriteCloser) (io.ReadWriteCloser, error)
}

func parseInt64(val reflect.Value) (int64, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return int64(val.Float()), nil
	case reflect.String:
		return strconv.ParseInt(val.String(), 10, 64)
	case reflect.Bool:
		if val.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		return 0, errors.New("syntax error")
	}
}

func parseUint64(val reflect.Value) (uint64, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return uint64(val.Float()), nil
	case reflect.String:
		return strconv.ParseUint(val.String(), 10, 64)
	case reflect.Bool:
		if val.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		return 0, errors.New("syntax error")
	}
}

func parseFloat64(val reflect.Value) (float64, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return val.Float(), nil
	case reflect.String:
		return strconv.ParseFloat(val.String(), 64)
	case reflect.Bool:
		if val.Bool() {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		return 0, errors.New("syntax error")
	}
}

func parseString(val reflect.Value) (string, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', 6, 64), nil
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		if val.Bool() {
			return "true", nil
		} else {
			return "false", nil
		}
	default:
		return "", errors.New("syntax error")
	}
}

func parseBool(val reflect.Value) (bool, error) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() > 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() > 0, nil
	case reflect.Float32, reflect.Float64:
		return val.Float() > 0, nil
	case reflect.String:
		return val.String() == "", nil
	case reflect.Bool:
		return val.Bool(), nil
	default:
		return false, errors.New("syntax error")
	}
}

func Register(name string, config Config) error {
	lock.Lock()
	defer lock.Unlock()
	if _, ok := streams[name]; ok {
		return fmt.Errorf("duplicate stream named %s", name)
	}
	streams[name] = config
	return nil
}

func GetStream(name string, config map[string]interface{}) (Config, error) {
	lock.RLock()
	defer lock.RUnlock()
	if c, ok := streams[name]; ok {
		nc := c.DeepCopy()
		v := reflect.Indirect(reflect.ValueOf(nc))
		t := v.Type()
		for i:=0;i<v.NumField();i++ {
			tf := t.Field(i)
			vf := v.Field(i)
			if !vf.CanSet() {
				continue
			}

			key := ""
			if tag := tf.Tag.Get("subsurface"); tag != "" {
				key = tag
			} else {
				key = strings.ToLower(t.Name())
			}

			if val, ok := config[key]; ok {
				m := reflect.ValueOf(val)
				if m.Kind() == vf.Kind() {
					vf.Set(m)
					continue
				}

				switch vf.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					mv, err := parseInt64(m)
					if err != nil {
						return nil, err
					}
					vf.SetInt(mv)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					mv, err := parseUint64(m)
					if err != nil {
						return nil, err
					}
					vf.SetUint(mv)
				case reflect.Float32, reflect.Float64:
					mv, err := parseFloat64(m)
					if err != nil {
						return nil, err
					}
					vf.SetFloat(mv)
				case reflect.String:
					mv, err := parseString(m)
					if err != nil {
						return nil, err
					}
					vf.SetString(mv)
				case reflect.Bool:
					mv, err := parseBool(m)
					if err != nil {
						return nil, err
					}
					vf.SetBool(mv)
				default:
					return nil, errors.New("wrong type config")
				}
			}
		}
		return nc, nil
	}
	return nil, fmt.Errorf("stream %s not found", name)
}
