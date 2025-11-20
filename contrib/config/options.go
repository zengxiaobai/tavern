package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Decoder is config decoder.
type Decoder func(*KeyValue, map[string]any) error

// Resolver resolve placeholder in config.
type Resolver func(map[string]any) error

// Merge is config merge func.
type Merge func(dst, src any) error

// Option is config option.
type Option func(*options)

type options struct {
	sources  []Source
	decoder  Decoder
	resolver Resolver
	merge    Merge
}

// WithSource with config source.
func WithSource(s ...Source) Option {
	return func(o *options) {
		o.sources = s
	}
}

// WithDecoder with config decoder.
// DefaultDecoder behavior:
// If KeyValue.Format is non-empty, then KeyValue.Value will be deserialized into map[string]any
// and stored in the config cache(map[string]any)
// if KeyValue.Format is empty,{KeyValue.Key : KeyValue.Value} will be stored in config cache(map[string]any)
func WithDecoder(d Decoder) Option {
	return func(o *options) {
		o.decoder = d
	}
}

// WithResolver with config resolver.
func WithResolver(r Resolver) Option {
	return func(o *options) {
		o.resolver = r
	}
}

// WithMergeFunc with config merge func.
func WithMergeFunc(m Merge) Option {
	return func(o *options) {
		o.merge = m
	}
}

// defaultDecoder decode config from source KeyValue
// to target map[string]any using src.Format codec.
func defaultDecoder(src *KeyValue, target map[string]any) error {
	if src.Format == "" {
		// expand key "aaa.bbb" into map[aaa]map[bbb]any
		keys := strings.Split(src.Key, ".")
		for i, k := range keys {
			if i == len(keys)-1 {
				target[k] = src.Value
			} else {
				sub := make(map[string]any)
				target[k] = sub
				target = sub
			}
		}
		return nil
	}
	if unmarshal := toUnmarshal(src.Format); unmarshal != nil {
		return unmarshal(src.Value, &target)
	}
	return fmt.Errorf("unsupported key: %s format: %s", src.Key, src.Format)
}

func expand(s string, mapping func(string) string) string {
	r := regexp.MustCompile(`\${(.*?)}`)
	re := r.FindAllStringSubmatch(s, -1)
	for _, i := range re {
		if len(i) == 2 { //nolint:gomnd
			s = strings.ReplaceAll(s, i[0], mapping(i[1]))
		}
	}
	return s
}

type Unmarshal func(data []byte, v any) error

func toUnmarshal(format string) Unmarshal {
	switch format {
	case "yaml", "yml":
		return yaml.Unmarshal
	default:
		return json.Unmarshal
	}
}
