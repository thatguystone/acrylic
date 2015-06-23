package acrylib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/DisposaBoy/JsonConfigReader"
	"gopkg.in/yaml.v2"
)

type meta map[string]interface{}

type metaType int

const (
	metaYaml metaType = iota
	metaToml
	metaJson
	metaUnknown
)

func (m *meta) merge(b []byte, mt metaType) error {
	switch mt {
	case metaYaml:
		return yaml.Unmarshal(b, m)
	case metaToml:
		return toml.Unmarshal(b, m)
	case metaJson:
		r := JsonConfigReader.New(bytes.NewReader(b))
		dec := json.NewDecoder(r)
		return dec.Decode(m)

	default:
		return fmt.Errorf("unrecognized meta type: %d", mt)
	}

}

func (m *meta) has() bool {
	return len(*m) > 0
}

func (m meta) getString(k string) string {
	s, ok := m[k].(string)
	if !ok {
		return ""
	}

	return s
}

func (m meta) getBool(k string) bool {
	s, ok := m[k].(bool)
	if !ok {
		return false
	}

	return s
}

func (m meta) getDate(k string) (time.Time, bool) {
	s := m.getString(k)
	return sToDate(s)
}

func (m meta) title() string {
	return m.getString("title")
}

func (m meta) date() (time.Time, bool) {
	return m.getDate("date")
}

func (m meta) summary() string {
	return m.getString("summary")
}

func (m meta) rss() bool {
	return m.getBool("rss")
}

func (m meta) layoutName() string {
	return m.getString("layoutName")
}

func (m meta) publish() (bool, bool) {
	b, ok := m["publish"].(bool)
	if !ok {
		return true, false
	}

	return b, true
}

func (m meta) menu() interface{} {
	return m["menu"]
}

func metaTypeFromString(t string) metaType {
	switch t {
	case "toml":
		return metaToml
	case "yaml":
		return metaYaml
	case "json":
		return metaJson
	default:
		return metaUnknown
	}
}
