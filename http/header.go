package http

import (
	"fmt"
	"io"
	"net/textproto"
	"sort"
)

// Header are HTTP headers.
type Header map[string][]string

// Get returns the first value for the given key.
func (h Header) Get(key string) string {
	return textproto.MIMEHeader(h).Get(key)
}

// Set sets the key value pair on the header.
func (h Header) Set(key, value string) {
	textproto.MIMEHeader(h).Set(key, value)
}

// Del deletes the header with the given key.
func (h Header) Del(key string) {
	textproto.MIMEHeader(h).Del(key)
}

func (h Header) Write(w io.Writer) error {
	kvs := sortedKeyValues(h)

	for _, kv := range kvs {
		for _, v := range kv.values {
			_, err := fmt.Fprintf(w, "%s: %s\r\n", kv.key, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type keyValue struct {
	key    string
	values []string
}

type keyValues []keyValue

func (k keyValues) Len() int           { return len(k) }
func (k keyValues) Less(i, j int) bool { return k[i].key < k[j].key }
func (k keyValues) Swap(i, j int)      { k[i], k[j] = k[j], k[i] }

func sortedKeyValues(header Header) keyValues {
	kvs := make(keyValues, 0, len(header))
	for k, v := range header {
		kvs = append(kvs, keyValue{key: k, values: v})
	}

	sort.Sort(kvs)
	return kvs
}
