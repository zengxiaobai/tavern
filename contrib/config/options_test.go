package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultDecoder(t *testing.T) {
	src := &KeyValue{
		Key:    "service",
		Value:  []byte("config"),
		Format: "",
	}
	target := make(map[string]interface{})
	err := defaultDecoder(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(target, map[string]interface{}{"service": []byte("config")}) {
		t.Fatal(`target is not equal to map[string]interface{}{"service": "config"}`)
	}

	src = &KeyValue{
		Key:    "service.name.alias",
		Value:  []byte("2233"),
		Format: "",
	}
	target = make(map[string]interface{})
	err = defaultDecoder(src, target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(map[string]interface{}{
		"service": map[string]interface{}{
			"name": map[string]interface{}{
				"alias": []byte("2233"),
			},
		},
	}, target) {
		t.Fatal(`target is not equal to map[string]interface{}{"service": map[string]interface{}{"name": map[string]interface{}{"alias": []byte("2233")}}}`)
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		input   string
		mapping func(string) string
		want    string
	}{
		{
			input: "${a}",
			mapping: func(s string) string {
				return strings.ToUpper(s)
			},
			want: "A",
		},
		{
			input: "a",
			mapping: func(s string) string {
				return strings.ToUpper(s)
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		if got := expand(tt.input, tt.mapping); got != tt.want {
			t.Errorf("expand() want: %s, got: %s", tt.want, got)
		}
	}
}

func TestWithMergeFunc(t *testing.T) {
	c := &options{}
	a := func(dst, src interface{}) error {
		return nil
	}
	WithMergeFunc(a)(c)
	if c.merge == nil {
		t.Fatal("c.merge is nil")
	}
}
