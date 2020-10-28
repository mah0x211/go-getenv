package getenv

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getFuncPointer(f interface{}) (uintptr, bool) {
	v := reflect.ValueOf(f)
	if v.Kind() != reflect.Func {
		return 0, false
	}
	return reflect.Indirect(v).Pointer(), true
}

func notEqualFuncs(t *testing.T, a interface{}, b interface{}) bool {
	ap, ok := getFuncPointer(a)
	if !ok {
		return false
	}
	bp, ok := getFuncPointer(b)
	if !ok {
		return false
	}
	return assert.NotEqual(t, ap, bp)
}

func equalFuncs(t *testing.T, a interface{}, b interface{}) bool {
	ap, ok := getFuncPointer(a)
	if !ok {
		return false
	}
	bp, ok := getFuncPointer(b)
	if !ok {
		return false
	}
	return assert.Equal(t, ap, bp)
}

func TestSet(t *testing.T) {
	defer func() {
		name2envs = map[string][]*Env{}
	}()

	parsefn := func(iv interface{}, k, v string) error {
		return nil
	}

	// test that set env
	strv := "string"
	bolv := true
	uipv := uintptr(123)
	intv := int(456)
	uintv := uint(789)
	f32v := float32(10.1)
	f64v := float64(11.2)
	for name, v := range map[string][]interface{}{
		"STR":     []interface{}{strv, &strv, nil},
		"BOL":     []interface{}{bolv, &bolv, parsefn},
		"UINTPTR": []interface{}{uipv, &uipv, nil},
		"INT":     []interface{}{intv, &intv, parsefn},
		"UINT":    []interface{}{uintv, &uintv, nil},
		"FLOAT32": []interface{}{f32v, &f32v, parsefn},
		"FLOAT64": []interface{}{f64v, &f64v, nil},
	} {
		desc := fmt.Sprintf("test %T env", v[0])
		if fn, ok := v[2].(ParseFunc); ok {
			assert.NoError(t, Set(name, desc, v[1], fn))
		} else {
			// use defaultParseFunc
			assert.NoError(t, Set(name, desc, v[1], nil))
			v[2] = defaultParseFunc
		}
		// confirm
		envs, ok := name2envs[name]
		assert.True(t, ok)
		assert.Len(t, envs, 1)
		env := envs[0]
		assert.Equal(t, name, env.Name)
		assert.Equal(t, desc, env.Description)
		assert.Equal(t, v[1], env.Value)
		assert.Equal(t, v[0], env.DefaultValue)
		equalFuncs(t, v[2], env.Parse)
	}

	// test that returns ErrName
	for _, name := range []string{
		"", "0BAR", " BAR", "BAR ", "BAR-BAZ",
	} {
		assert.Equal(t, ErrName, Set(name, "", nil, nil))
	}

	// test that returns ErrValue
	for _, v := range []interface{}{
		strv, bolv, uipv, intv, uintv, f32v, f64v,
		nil,
		[]string{},
		map[string]string{},
		struct{}{},
		&[]string{},
		&map[string]string{},
		&struct{}{},
	} {
		assert.Equal(t, ErrValue, Set("BAR", "", v, nil))
	}

	// test that returns error
	for name, v := range map[string]interface{}{
		"STR":     &strv,
		"BOL":     &bolv,
		"UINTPTR": &uipv,
		"INT":     &intv,
		"UINT":    &uintv,
		"FLOAT32": &f32v,
		"FLOAT64": &f64v,
	} {
		err := Set(name+"_ADD", "", v, nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrValueInUsed))
		fmt.Printf("%v\n", err)
	}
}

func TestUsage(t *testing.T) {
	defer func() {
		name2envs = map[string][]*Env{}
	}()

	// setup
	strv := "string"
	bolv := true
	uipv := uintptr(123)
	intv := int(456)
	uintv := uint(789)
	f32v := float32(10.1)
	f64v := float64(11.2)
	vals := map[string][]interface{}{
		"STR":     []interface{}{strv, &strv},
		"BOL":     []interface{}{bolv, &bolv},
		"UINTPTR": []interface{}{uipv, &uipv},
		"INT":     []interface{}{intv, &intv},
		"UINT":    []interface{}{uintv, &uintv},
		"FLOAT32": []interface{}{f32v, &f32v},
		"FLOAT64": []interface{}{f64v, &f64v},
	}
	names := make([]string, 0, len(vals))
	for name, v := range vals {
		desc := fmt.Sprintf("test %q env", name)
		assert.NoError(t, Set(name, desc, v[1], nil))
		names = append(names, name)
	}
	sort.Strings(names)

	// test that set sort by name
	Usage(func(name, desc string, defval interface{}) {
		assert.Equal(t, names[0], name)
		assert.Equal(t, fmt.Sprintf("test %q env", name), desc)
		assert.Equal(t, vals[name][0], defval)
		names = names[1:]
	})
}

func TestParse(t *testing.T) {
	defer func() {
		name2envs = map[string][]*Env{}
	}()

	// setup
	suffix := "_" + strconv.FormatInt(time.Now().Unix(), 10)
	ncall := 0
	var parseErr error
	parsefn := func(iv interface{}, k, v string) error {
		ncall++
		return parseErr
	}
	strv := "string"
	bolv := true
	uipv := uintptr(123)
	intv := int(456)
	uintv := uint(789)
	f32v := float32(10.1)
	f64v := float64(11.2)
	vals := map[string][]interface{}{
		"STR" + suffix:     []interface{}{strv, &strv, "env string"},
		"BOL" + suffix:     []interface{}{bolv, &bolv, false},
		"UINTPTR" + suffix: []interface{}{uipv, &uipv, uintptr(321)},
		"INT" + suffix:     []interface{}{intv, &intv, int(654)},
		"UINT" + suffix:    []interface{}{uintv, &uintv, uint(987)},
		"FLOAT32" + suffix: []interface{}{f32v, &f32v, float32(1.01)},
		"FLOAT64" + suffix: []interface{}{f64v, &f64v, float64(2.11)},
	}
	envnames := make([]string, 0, len(vals))
	for name, v := range vals {
		envnames = append(envnames, name)
		assert.NoError(t, Set(name, "", v[1], parsefn))
	}
	defer func() {
		for _, name := range envnames {
			os.Unsetenv(name)
		}
	}()

	// test that not call parser
	assert.NoError(t, Parse())
	assert.Equal(t, 0, ncall)

	// test that calling parser if the environment variables defined
	n := 0
	for name, v := range vals {
		envval := fmt.Sprintf("%v", v[2])
		os.Setenv(name, envval)
		n++
		assert.NoError(t, Parse())
		assert.Equal(t, n, ncall)
		ncall = 0
		env := name2envs[name][0]
		assert.Equal(t, env.DefaultValue, v[0])
	}

	// test that stops parsing and returns error when the parser returns an error
	parseErr = fmt.Errorf("custom parse error")
	err := Parse()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom parse error")
	assert.Equal(t, 1, ncall)

	// test that use defaultParseFunc if parser is not defined
	name2envs = map[string][]*Env{}
	for name, v := range vals {
		envnames = append(envnames, name)
		assert.NoError(t, Set(name, "", v[1], nil))
	}
	assert.NoError(t, Parse())
	for _, v := range vals {
		exp := v[2]
		act := reflect.Indirect(reflect.ValueOf(v[1])).Interface()
		assert.Equal(t, fmt.Sprintf("%v", exp), fmt.Sprintf("%v", act))
	}

	// test that returns ErrEnvVar if cannot convert environment variable to actual value
	envval := "{{unparsable env value}}"
	var envname string
	for _, name := range envnames {
		if !strings.HasPrefix(name, "STR") {
			os.Setenv(name, envval)
			envname = name
			break
		}
	}
	err = Parse()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), envname)
	assert.Contains(t, err.Error(), envval)
	assert.True(t, errors.Is(err, ErrEnvVar))
}
