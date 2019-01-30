package parser

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

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

func Unmarshal(tagName string, val reflect.Value, data map[string]interface{}) error {
	val = reflect.Indirect(val)
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		tf := typ.Field(i)
		vf := val.Field(i)
		if !vf.CanSet() {
			continue
		}

		key := ""
		if tag := tf.Tag.Get(tagName); tagName != "" && tag != "" {
			key = tag
		} else {
			key = strings.ToLower(typ.Name())
		}

		if d, ok := data[key]; ok {
			m := reflect.ValueOf(d)
			if m.Kind() == vf.Kind() {
				vf.Set(m)
				continue
			}

			switch vf.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				mv, err := parseInt64(m)
				if err != nil {
					return err
				}
				vf.SetInt(mv)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				mv, err := parseUint64(m)
				if err != nil {
					return err
				}
				vf.SetUint(mv)
			case reflect.Float32, reflect.Float64:
				mv, err := parseFloat64(m)
				if err != nil {
					return err
				}
				vf.SetFloat(mv)
			case reflect.String:
				mv, err := parseString(m)
				if err != nil {
					return err
				}
				vf.SetString(mv)
			case reflect.Bool:
				mv, err := parseBool(m)
				if err != nil {
					return err
				}
				vf.SetBool(mv)
			case reflect.Struct:
				d, ok := d.(map[string]interface{})
				if ok {
					if err := Unmarshal(tagName, val, d); err != nil {
						return err
					}
				}
			default:
				return errors.New("wrong type config")
			}
		}
	}
	return nil
}

