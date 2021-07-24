package getenv

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func isUpper(b byte) bool {
	return 'A' <= b && b <= 'Z'
}

func isLower(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isAlpha(b byte) bool {
	return isUpper(b) || isLower(b)
}

func isFunc(fn interface{}) bool {
	v := reflect.ValueOf(fn)
	return v.Kind() == reflect.Func && !v.IsNil()
}

func parseInt(s string, k reflect.Kind) (int64, error) {
	switch k {
	case reflect.Int:
		return strconv.ParseInt(s, 10, 0)
	case reflect.Int8:
		return strconv.ParseInt(s, 10, 8)
	case reflect.Int16:
		return strconv.ParseInt(s, 10, 16)
	case reflect.Int32:
		return strconv.ParseInt(s, 10, 32)
	case reflect.Int64:
		return strconv.ParseInt(s, 10, 64)
	default:
		panic(fmt.Errorf("bug: unsupported integer types %v", k))
	}
}

func parseUint(s string, k reflect.Kind) (uint64, error) {
	switch k {
	case reflect.Uint:
		return strconv.ParseUint(s, 10, 0)
	case reflect.Uint8:
		return strconv.ParseUint(s, 10, 8)
	case reflect.Uint16:
		return strconv.ParseUint(s, 10, 16)
	case reflect.Uint32:
		return strconv.ParseUint(s, 10, 32)
	case reflect.Uint64, reflect.Uintptr:
		return strconv.ParseUint(s, 10, 64)
	default:
		panic(fmt.Errorf("bug: unsupported unsigned integer types %v", k))
	}
}

func parseFloat(s string, k reflect.Kind) (float64, error) {
	switch k {
	case reflect.Float32:
		return strconv.ParseFloat(s, 32)
	case reflect.Float64:
		return strconv.ParseFloat(s, 64)
	default:
		panic(fmt.Errorf("bug: unsupported float types %v", k))
	}
}

type ParseFunc func(iv interface{}, envName, envValue string) error

func defaultParseFunc(iv interface{}, envName, envValue string) error {
	ref := reflect.ValueOf(iv)
	if ref.Kind() != reflect.Ptr {
		return ErrValue
	}

	ref = reflect.Indirect(ref)
	kind := ref.Kind()
	switch kind {
	case reflect.String:
		ref.SetString(envValue)

	case reflect.Bool:
		v, err := strconv.ParseBool(envValue)
		if err != nil {
			return err
		}
		ref.SetBool(v)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := parseInt(envValue, kind)
		if err != nil {
			return err
		}
		ref.SetInt(v)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		v, err := parseUint(envValue, kind)
		if err != nil {
			return err
		}
		ref.SetUint(v)

	case reflect.Float32, reflect.Float64:
		v, err := parseFloat(envValue, kind)
		if err != nil {
			return err
		}
		ref.SetFloat(v)

	default:
		panic(fmt.Errorf("bug: unsupported value types %v", kind))
	}

	return nil
}

type CheckFunc func(iv interface{}, envName string) error

func defaultCheckFunc(iv interface{}, envName string) error {
	// allow any value
	return nil
}

var ErrName = fmt.Errorf("name must be non-empty printable ascii string and that must not contain spaces and '='")

func checkName(s string) error {
	n := len(s)
	// must be non-empty string
	if n == 0 {
		return ErrName
	}

	// first character must be [A-Za-z_]
	c := s[0]
	if !isAlpha(c) && c != '_' {
		return ErrName
	}
	// following characters must be [0-9A-Za-z_]
	for i := 1; i < n; i++ {
		c := s[i]
		if isAlpha(c) || isDigit(c) || c == '_' {
			continue
		}
		return ErrName
	}

	return nil
}

var ErrValue = fmt.Errorf("value must be non-nil pointer of following types: string, bool, uintptr, 8-64 bit int or uint and 32-64 bit float")

func checkValue(v interface{}) (interface{}, error) {
	ref := reflect.ValueOf(v)
	if ref.Kind() != reflect.Ptr {
		return nil, ErrValue
	}

	ref = reflect.Indirect(ref)
	switch ref.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Uint, reflect.Uintptr,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return ref.Interface(), nil
	}

	return nil, ErrValue
}

type Env struct {
	Name         string
	Description  string
	DefaultValue interface{}
	Value        interface{}
	Parse        ParseFunc
	Check        CheckFunc
}

var name2envs = map[string][]*Env{}

var ErrValueInUsed = fmt.Errorf("value already in used")

func Set(name, desc string, value interface{}, parsefn ParseFunc, checkfn CheckFunc) error {
	var defval interface{}
	// check arguments
	if err := checkName(name); err != nil {
		return err
	} else if defval, err = checkValue(value); err != nil {
		return err
	}
	if !isFunc(parsefn) {
		parsefn = defaultParseFunc
	}
	if !isFunc(checkfn) {
		checkfn = defaultCheckFunc
	}

	// check that the value is already use by another names
	for k, envs := range name2envs {
		for _, env := range envs {
			if env.Value == value {
				return fmt.Errorf("%w: %p already use by %q", ErrValueInUsed, value, k)
			}
		}
	}

	// set new env
	envs, ok := name2envs[name]
	if !ok {
		envs = []*Env{}
	}
	name2envs[name] = append(envs, &Env{
		Name:         name,
		Description:  desc,
		DefaultValue: defval,
		Value:        value,
		Parse:        parsefn,
		Check:        checkfn,
	})

	return nil
}

type UsageFunc func(name, desc string, defval interface{})

func Usage(usagefn UsageFunc) {
	names := make([]string, 0, len(name2envs))
	for name := range name2envs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		envs := name2envs[name]
		for _, env := range envs {
			usagefn(name, env.Description, env.DefaultValue)
		}
	}
}

var ErrEnvVar = fmt.Errorf("invalid environment variable")

func Parse() error {
	for name, envs := range name2envs {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			for _, env := range envs {
				if err := env.Parse(env.Value, name, v); err != nil {
					return fmt.Errorf("%w: %q %v", ErrEnvVar, name, err)
				} else if err = env.Check(env.Value, name); err != nil {
					return fmt.Errorf("%w: %q %v", ErrEnvVar, name, err)
				}
			}
		}
	}
	return nil
}
