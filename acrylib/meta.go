package acrylib

import (
	"time"

	"gopkg.in/yaml.v2"
)

type meta map[string]interface{}

func (m *meta) merge(b []byte) error {
	return yaml.Unmarshal(b, m)
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
