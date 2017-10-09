package log

// Field represents a giving map of values associated with a giving field value.
type Field map[string]interface{}

// GetBool collects the string value of a key if it exists.
func (p Field) GetBool(key string) (bool, bool) {
	val, found := p.Get(key)
	if !found {
		return false, false
	}

	value, ok := val.(bool)
	return value, ok
}

// GetFloat64 collects the string value of a key if it exists.
func (p Field) GetFloat64(key string) (float64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float64)
	return value, ok
}

// GetFloat32 collects the string value of a key if it exists.
func (p Field) GetFloat32(key string) (float32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(float32)
	return value, ok
}

// GetInt8 collects the string value of a key if it exists.
func (p Field) GetInt8(key string) (int8, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int8)
	return value, ok
}

// GetInt16 collects the string value of a key if it exists.
func (p Field) GetInt16(key string) (int16, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int16)
	return value, ok
}

// GetInt64 collects the string value of a key if it exists.
func (p Field) GetInt64(key string) (int64, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int64)
	return value, ok
}

// GetInt32 collects the string value of a key if it exists.
func (p Field) GetInt32(key string) (int32, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int32)
	return value, ok
}

// GetInt collects the string value of a key if it exists.
func (p Field) GetInt(key string) (int, bool) {
	val, found := p.Get(key)
	if !found {
		return 0, false
	}

	value, ok := val.(int)
	return value, ok
}

// GetString collects the string value of a key if it exists.
func (p Field) GetString(key string) (string, bool) {
	val, found := p.Get(key)
	if !found {
		return "", false
	}

	value, ok := val.(string)
	return value, ok
}

// Get collects the value of a key if it exists.
func (p Field) Get(key string) (value interface{}, found bool) {
	if p == nil {
		return
	}

	val, ok := p[key]
	return val, ok
}
