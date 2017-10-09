package log

// NilPair defines a nil starting pair.
var NilPair = (*Pair)(nil)

// Pair defines a struct for storing a linked pair of key and values.
type Pair struct {
	prev  *Pair
	key   string
	value interface{}
}

// NewPair returns a a key-value pair chain for setting fields.
func NewPair(key string, value interface{}) *Pair {
	return &Pair{
		key:   key,
		value: value,
	}
}

// Append returns a new Pair with the giving key and with the provded Pair set as
// it's previous link.
func Append(p *Pair, key string, value interface{}) *Pair {
	return p.Append(key, value)
}

// Fields returns all internal pair data as a map.
func (p *Pair) Fields() Field {
	if p == nil {
		return make(Field)
	}

	if p.prev == nil {
		f := make(Field)
		f[p.key] = p.value
		return f
	}

	f := p.prev.Fields()
	if p.key != "" {
		f[p.key] = p.value
	}

	return f
}

// Append returns a new pair with the giving key and value and its previous
// set to this pair.
func (p *Pair) Append(key string, val interface{}) *Pair {
	return &Pair{
		prev:  p,
		key:   key,
		value: val,
	}
}

// Root returns the root Pair in the chain which links all pairs together.
func (p *Pair) Root() *Pair {
	if p.prev == nil {
		return p
	}

	return p.prev.Root()
}

// GetBool collects the string value of a key if it exists.
func (p *Pair) GetBool(key string) (bool, bool) {
	val, found := p.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 collects the string value of a key if it exists.
func (p *Pair) GetFloat64(key string) (float64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 collects the string value of a key if it exists.
func (p *Pair) GetFloat32(key string) (float32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 collects the string value of a key if it exists.
func (p *Pair) GetInt8(key string) (int8, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 collects the string value of a key if it exists.
func (p *Pair) GetInt16(key string) (int16, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 collects the string value of a key if it exists.
func (p *Pair) GetInt64(key string) (int64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 collects the string value of a key if it exists.
func (p *Pair) GetInt32(key string) (int32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt collects the string value of a key if it exists.
func (p *Pair) GetInt(key string) (int, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString collects the string value of a key if it exists.
func (p *Pair) GetString(key string) (string, bool) {
	val, found := p.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

// Get collects the value of a key if it exists.
func (p *Pair) Get(key string) (value interface{}, found bool) {
	if p == nil {
		return
	}

	if p.key == key {
		return p.value, true
	}

	if p.prev == nil {
		return
	}

	return p.prev.Get(key)
}
