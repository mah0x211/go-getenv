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
		name2envs = map[string]*Env{}
	}()

	parsefn := func(iv interface{}, k, v string) error {
		return nil
	}

	checkfn := func(iv interface{}, k string) error {
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
		"STR":     {strv, &strv, nil, nil},
		"BOL":     {bolv, &bolv, parsefn, checkfn},
		"UINTPTR": {uipv, &uipv, nil, nil},
		"INT":     {intv, &intv, parsefn, checkfn},
		"UINT":    {uintv, &uintv, nil, nil},
		"FLOAT32": {f32v, &f32v, parsefn, checkfn},
		"FLOAT64": {f64v, &f64v, nil, nil},
	} {
		desc := fmt.Sprintf("test %T env", v[0])
		if fn, ok := v[2].(ParseFunc); ok {
			assert.NoError(t, Set(name, desc, v[1], false, fn, v[3].(CheckFunc)))
		} else {
			// use defaultParseFunc
			assert.NoError(t, Set(name, desc, v[1], false, nil, nil))
			v[2] = defaultParseFunc
		}
		// confirm
		env, ok := name2envs[name]
		assert.True(t, ok)
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
		assert.Equal(t, ErrName, Set(name, "", nil, false, nil, nil))
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
		assert.Equal(t, ErrValue, Set("BAR", "", v, false, nil, nil))
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
		err := Set(name, "", v, false, nil, nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNameAlready))
		fmt.Printf("%v\n", err)
	}
}

func TestUsage(t *testing.T) {
	defer func() {
		name2envs = map[string]*Env{}
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
		"STR":     {strv, &strv},
		"BOL":     {bolv, &bolv},
		"UINTPTR": {uipv, &uipv},
		"INT":     {intv, &intv},
		"UINT":    {uintv, &uintv},
		"FLOAT32": {f32v, &f32v},
		"FLOAT64": {f64v, &f64v},
	}
	names := make([]string, 0, len(vals))
	for name, v := range vals {
		desc := fmt.Sprintf("test %q env", name)
		assert.NoError(t, Set(name, desc, v[1], true, nil, nil))
		names = append(names, name)
	}
	sort.Strings(names)

	// test that set sort by name
	Usage(func(name, desc string, defval interface{}, required bool) {
		assert.Equal(t, names[0], name)
		assert.Equal(t, fmt.Sprintf("test %q env", name), desc)
		assert.Equal(t, vals[name][0], defval)
		assert.True(t, required)
		names = names[1:]
	})
}

func TestParse(t *testing.T) {
	defer func() {
		name2envs = map[string]*Env{}
	}()

	// setup
	suffix := "_" + strconv.FormatInt(time.Now().Unix(), 10)
	var nCallParseFn int
	var parseErr error
	parsefn := func(iv interface{}, k, v string) error {
		nCallParseFn++
		return parseErr
	}

	var nCallCheckFn int
	var checkErr error
	checkfn := func(iv interface{}, k string) error {
		nCallCheckFn++
		return checkErr
	}

	strv := "string"
	bolv := true
	uipv := uintptr(123)
	intv := int(456)
	uintv := uint(789)
	f32v := float32(10.1)
	f64v := float64(11.2)
	vals := map[string][]interface{}{
		"STR" + suffix:     {strv, &strv, "env string"},
		"BOL" + suffix:     {bolv, &bolv, false},
		"UINTPTR" + suffix: {uipv, &uipv, uintptr(321)},
		"INT" + suffix:     {intv, &intv, int(654)},
		"UINT" + suffix:    {uintv, &uintv, uint(987)},
		"FLOAT32" + suffix: {f32v, &f32v, float32(1.01)},
		"FLOAT64" + suffix: {f64v, &f64v, float64(2.11)},
	}
	envnames := make([]string, 0, len(vals))
	for name, v := range vals {
		envnames = append(envnames, name)
		assert.NoError(t, Set(name, "", v[1], false, parsefn, checkfn))
	}
	defer func() {
		for _, name := range envnames {
			os.Unsetenv(name)
		}
	}()

	// test that not call parser and checker
	assert.NoError(t, Parse())
	assert.Equal(t, 0, nCallParseFn)
	assert.Equal(t, 0, nCallCheckFn)

	// test that calling parser if the environment variables defined
	n := 0
	for name, v := range vals {
		envval := fmt.Sprintf("%v", v[2])
		os.Setenv(name, envval)
		n++
		assert.NoError(t, Parse())
		assert.Equal(t, n, nCallParseFn)
		nCallParseFn = 0
		assert.Equal(t, n, nCallCheckFn)
		nCallCheckFn = 0
		env := name2envs[name]
		assert.Equal(t, env.DefaultValue, v[0])
	}

	// test that stops parsing and returns error when the parser returns an error
	parseErr = fmt.Errorf("custom parse error")
	err := Parse()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom parse error")
	assert.Equal(t, 1, nCallParseFn)
	assert.Equal(t, 0, nCallCheckFn)

	// test that stops parsing and returns error when the checker returns an error
	parseErr = nil
	nCallParseFn = 0
	checkErr = fmt.Errorf("custom check error")
	err = Parse()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom check error")
	assert.Equal(t, 1, nCallParseFn)
	assert.Equal(t, 1, nCallCheckFn)

	// test that use defaultParseFunc if parser is not defined
	name2envs = map[string]*Env{}
	nCallParseFn = 0
	nCallCheckFn = 0
	checkErr = nil
	for name, v := range vals {
		envnames = append(envnames, name)
		assert.NoError(t, Set(name, "", v[1], false, nil, checkfn))
	}
	assert.NoError(t, Parse())
	assert.Equal(t, 0, nCallParseFn)
	assert.Equal(t, len(vals), nCallCheckFn)

	// test that use defaultCheckFunc if checker is not defined
	name2envs = map[string]*Env{}
	nCallParseFn = 0
	nCallCheckFn = 0
	for name, v := range vals {
		envnames = append(envnames, name)
		assert.NoError(t, Set(name, "", v[1], false, parsefn, nil))
	}
	assert.NoError(t, Parse())
	assert.Equal(t, len(vals), nCallParseFn)
	assert.Equal(t, 0, nCallCheckFn)
	for _, v := range vals {
		exp := v[2]
		act := reflect.Indirect(reflect.ValueOf(v[1])).Interface()
		assert.Equal(t, fmt.Sprintf("%v", exp), fmt.Sprintf("%v", act))
	}

	// test that returns ErrEnvVar if cannot convert environment variable to actual value
	name2envs = map[string]*Env{}
	envname := ""
	envval := "{{unparsable env value}}"
	for name, v := range vals {
		if !strings.HasPrefix(name, "STR") {
			envname = name
			os.Setenv(name, envval)
			assert.NoError(t, Set(name, "", v[1], false, nil, nil))
			break
		}
	}
	err = Parse()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrEnvVar))
	assert.Contains(t, err.Error(), envname)
	assert.Contains(t, err.Error(), envval)

	// test that returns ErrNotDefined if environment variable is not defined
	name2envs = map[string]*Env{}
	for name, v := range vals {
		if !strings.HasPrefix(name, "STR") {
			os.Unsetenv(name)
			assert.NoError(t, Set(name, "", v[1], true, nil, nil))
			break
		}
	}
	err = Parse()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotDefined))
}
