package varycontrol

import (
	"net/http"
	"slices"
	"sort"
	"strings"
)

const (
	VaryEmptyIdentity = "tr_identity"
)

type Key []string

func (k *Key) String() string {
	return strings.Join(*k, ",")
}

func (k *Key) Append(val string) {
	keys := canonical(val)
	if len(keys) > 0 {
		if slices.Contains(*k, val) {
			return
		}
		*k = append(*k, keys...)
	}
}

func (k *Key) VaryData(h http.Header) string {
	l := len(*k)
	if l <= 0 {
		return ""
	}

	kv := make(map[string]string, l)
	for _, key := range *k {
		v := normalizeHeaderValue(key, h.Values(key))
		kv[key] = v
	}

	keys := make([]string, 0, len(kv))
	for key := range kv {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var buf strings.Builder
	for _, key := range keys {
		v := kv[key]
		if buf.Len() > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(v)
	}
	return buf.String()
}

func Clean(values ...string) Key {
	keys := make([]string, 0)

	for _, val := range values {
		key := canonical(val)
		if len(key) > 0 {
			keys = append(keys, key...)
		}
	}

	sort.Strings(keys)

	return slices.Compact(keys)
}

func canonical(val string) []string {
	s := strings.TrimSpace(val)
	if s == "" || s == "," {
		return nil
	}

	keys := make([]string, 0)

	vk := strings.Split(s, ",")
	for _, k := range vk {
		key := strings.TrimSpace(k)
		if key != "" {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil
	}

	return keys
}

func sortValues(vals []string) string {
	if len(vals) == 1 {
		return sortValue(vals[0])
	}

	v := make([]string, 0, len(vals))
	for _, val := range vals {
		v = append(v, sortValue(val))
	}

	sort.Strings(v)

	return strings.Join(v, ",")
}

func sortValue(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}

	v := splitTrimSpace(val)

	sort.Strings(v)

	return strings.Join(v, ",")
}

func splitTrimSpace(s string) []string {
	if s == "" {
		return nil
	}

	v := strings.Split(s, ",")
	for i := range v {
		v[i] = strings.TrimSpace(v[i])
	}

	return v
}

func normalizeHeaderValue(key string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	keyLower := strings.ToLower(strings.TrimSpace(key))

	if keyLower == "accept-encoding" {
		return normalizeAcceptEncoding(values)
	}

	return sortValues(values)
}

func normalizeAcceptEncoding(values []string) string {
	if len(values) == 0 {
		return ""
	}

	encodings := make([]string, 0)
	for _, val := range values {
		parts := strings.Split(val, ",")
		for _, part := range parts {
			encoding := strings.TrimSpace(part)
			if encoding != "" {
				encodings = append(encodings, encoding)
			}
		}
	}

	if len(encodings) == 0 {
		return ""
	}

	sort.Strings(encodings)

	return strings.Join(encodings, ",")
}

func BuildVaryKeyForCache(varyHeaders []string, reqHeaders http.Header, respHeaders http.Header) string {
	if len(varyHeaders) == 0 {
		return ""
	}

	kv := make(map[string]string)
	for _, varyKey := range varyHeaders {
		key := strings.ToLower(strings.TrimSpace(varyKey))
		if key == "" {
			continue
		}

		if key == "accept-encoding" {
			contentEncoding := GetResponseContentEncodingFromHeaders(respHeaders)
			if contentEncoding != "" {
				kv[key] = NormalizeContentEncoding(contentEncoding)
			}
		} else {
			values := reqHeaders.Values(key)
			if len(values) > 0 {
				kv[key] = normalizeHeaderValue(key, values)
			}
		}
	}

	if len(kv) == 0 {
		return ""
	}

	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for i, key := range keys {
		if i > 0 {
			buf.WriteByte('&')
		}
		buf.WriteString(key)
		buf.WriteByte('=')
		buf.WriteString(kv[key])
	}

	return buf.String()
}

func GetResponseContentEncodingFromHeaders(headers http.Header) string {
	encoding := headers.Get("Content-Encoding")
	if encoding == "" {
		return ""
	}

	encodings := strings.Split(encoding, ",")
	if len(encodings) > 0 {
		return strings.TrimSpace(encodings[0])
	}
	return ""
}
