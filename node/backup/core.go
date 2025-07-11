package backup

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/status-im/status-go/eth-node/crypto"
)

type core struct {
	dumpers map[string]func() ([]byte, error)
	loaders map[string]func([]byte) error
}

func newCore() *core {
	return &core{
		dumpers: make(map[string]func() ([]byte, error)),
		loaders: make(map[string]func([]byte) error),
	}
}

func (c *core) Register(
	componentName string,
	dumpFunc func() ([]byte, error),
	loadFunc func([]byte) error,
) {
	c.dumpers[componentName] = dumpFunc
	c.loaders[componentName] = loadFunc
}

func (b *core) Create(privateKey []byte) ([]byte, error) {
	dumped, err := b.dump()
	if err != nil {
		return nil, fmt.Errorf("dump failed: %w", err)
	}

	data, err := marshal(dumped)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	encryptedData, err := crypto.EncryptSymmetric(privateKey, data)
	if err != nil {
		return nil, fmt.Errorf("encrypt failed: %w", err)
	}

	return encryptedData, nil
}

func (b *core) Restore(privateKey []byte, encrypted []byte) error {
	decrypted, err := crypto.DecryptSymmetric(privateKey, encrypted)
	if err != nil {
		return fmt.Errorf("decrypt failed: %w", err)
	}

	data, err := unmarshal(decrypted)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	err = b.load(data)
	if err != nil {
		return fmt.Errorf("load failed: %w", err)
	}

	return nil
}

func (b *core) dump() (map[string][]byte, error) {
	result := make(map[string][]byte)

	for name, fn := range b.dumpers {
		raw, err := fn()
		if err != nil {
			return nil, err
		}
		result[name] = raw
	}

	return result, nil
}

func (b *core) load(data map[string][]byte) error {
	for name, fn := range b.loaders {
		raw, ok := data[name]
		if !ok {
			continue
		}
		if err := fn(raw); err != nil {
			return fmt.Errorf("load %q failed: %w", name, err)
		}
	}

	return nil
}

func marshal(data map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshal(data []byte) (map[string][]byte, error) {
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)
	var result map[string][]byte
	err := dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
