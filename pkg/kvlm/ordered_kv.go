package kvlm

// OrderedKV keeps an ordered map
// because order matters in git
type OrderedKV struct {
	kv   map[string][]byte
	keys []string
}

func (okv *OrderedKV) Has(key string) bool {
	_, ok := okv.kv[key]
	return ok
}

func (okv *OrderedKV) Get(key string) ([]byte, bool) {
	val, ok := okv.kv[key]
	return val, ok
}

func (okv *OrderedKV) Set(key string, val []byte) {
	if value, ok := okv.kv[key]; ok {
		okv.kv[key] = append(value, val...)
	} else {
		okv.kv[key] = value
		okv.keys = append(okv.keys, key)
	}
}

func (okv *OrderedKV) Keys() []string {
	return okv.keys
}
