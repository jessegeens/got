package kvlm

import (
	"strings"
)

// TODO(jgeens): make output bytearray instead of string
func (kvlm *Kvlm) Serialize() string {
	var serialized string

	for _, k := range kvlm.Okv.Keys() {
		val, _ := kvlm.Okv.Get(k)

		line := k + " " + strings.Replace(string(val), "\n", "\n ", -1) + "\n"
		serialized = serialized + line
	}

	return serialized
}
