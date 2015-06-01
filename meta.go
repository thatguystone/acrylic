package toner

import "gopkg.in/yaml.v2"

type meta map[string]interface{}

func (m meta) merge(b []byte) error {
	return yaml.Unmarshal(b, &m)
}

func (m meta) publish() bool {
	b, ok := m["publish"].(bool)
	if !ok {
		return false
	}

	return b
}
