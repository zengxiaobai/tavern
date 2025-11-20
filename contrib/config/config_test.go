package config

import (
	"errors"
	"testing"
)

const (
	_testJSON = `{
	"system": {
		"pidfile": "/var/run/proxy.pid",
		"workdir": "/var/run/proxy",
		"user": "nobody",
		"log_encrypt": true
	},
   	"logger": {
		"level": "debug",
		"path": "/var/log/proxy"
	}
}`
)

type testConfigStruct struct {
	System struct {
		PidFile    string `json:"pidfile"`
		WorkDir    string `json:"workdir"`
		User       string `json:"user"`
		LogEncrypt bool   `json:"log_encrypt"`
	} `json:"system"`
	Logger struct {
		Level string `json:"level"`
		Path  string `json:"path"`
	} `json:"logger"`
}

type testJSONSource struct {
	data string
	sig  chan struct{}
	err  chan struct{}
}

func newTestJSONSource(data string) *testJSONSource {
	return &testJSONSource{data: data, sig: make(chan struct{}), err: make(chan struct{})}
}

func (p *testJSONSource) Load() ([]*KeyValue, error) {
	kv := &KeyValue{
		Key:    "json",
		Value:  []byte(p.data),
		Format: "json",
	}
	return []*KeyValue{kv}, nil
}

func (p *testJSONSource) Watch() (Watcher, error) {
	return newTestWatcher(p.sig, p.err), nil
}

type testWatcher struct {
	sig  chan struct{}
	err  chan struct{}
	exit chan struct{}
}

func newTestWatcher(sig, err chan struct{}) Watcher {
	return &testWatcher{sig: sig, err: err, exit: make(chan struct{})}
}

func (w *testWatcher) Next() ([]*KeyValue, error) {
	select {
	case <-w.sig:
		return nil, nil
	case <-w.err:
		return nil, errors.New("error")
	case <-w.exit:
		return nil, nil
	}
}

func (w *testWatcher) Stop() error {
	close(w.exit)
	return nil
}

func TestConfigNew(t *testing.T) {
	c := New[testConfigStruct](
		WithSource(newTestJSONSource(_testJSON)),
	)

	var bc testConfigStruct
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}

	if bc.System.PidFile != "/var/run/proxy.pid" {
		t.Error("pidfile error")
	}

	if bc.Logger.Level != "debug" {
		t.Error("level error")
	}
}
