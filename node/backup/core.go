package backup

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/status-im/status-go/crypto"
)

type core struct {
	backupProviders map[string]BackupProvider
}

func newCore() *core {
	return &core{
		backupProviders: make(map[string]BackupProvider),
	}
}

func (c *core) Register(
	componentName string,
	provider BackupProvider,
) {
	c.backupProviders[componentName] = provider
}

func (b *core) Create(privateKey []byte) ([]byte, error) {
	backup, err := b.exportBackup()
	if err != nil {
		return nil, fmt.Errorf("exportBackup failed: %w", err)
	}

	data, err := marshal(backup)
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

	err = b.importBackup(data)
	if err != nil {
		return fmt.Errorf("importBackup failed: %w", err)
	}

	return nil
}

func (b *core) exportBackup() (map[string][]byte, error) {
	result := make(map[string][]byte, len(b.backupProviders))

	for name, provider := range b.backupProviders {
		raw, err := provider.ExportBackup()
		if err != nil {
			return nil, err
		}
		result[name] = raw
	}

	return result, nil
}

func (b *core) importBackup(data map[string][]byte) error {
	var errs []error
	for name, provider := range b.backupProviders {
		raw, ok := data[name]
		if !ok {
			continue
		}
		if err := provider.ImportBackup(raw); err != nil {
			errs = append(errs, fmt.Errorf("importBackup %q failed: %w", name, err))
		}
	}

	return errors.Join(errs...)
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
