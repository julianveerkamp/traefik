package acme

import (
	"context"
	"encoding/json"

	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/traefik/traefik/v2/pkg/log"
)

var _ Store = (*ValkeyrieStore)(nil)

// ValkeyrieStore Stores implementation DynamoDB database.
type ValkeyrieStore struct {
	ctx context.Context
	kv  store.Store
}

// NewValkeyrieStore initializes a new ValkeyrieStore with an URL.
func NewValkeyrieStore(addr string, storeName string, config valkeyrie.Config) *ValkeyrieStore {
	logger := log.WithoutContext().WithField(log.ProviderName, "acme")
	ctx := context.Background()
	kv, err := valkeyrie.NewStore(ctx, storeName, []string{addr}, config)
	if err != nil {
		logger.Error(err)
	}
	s := &ValkeyrieStore{ctx: ctx, kv: kv}

	return s
}

func (s *ValkeyrieStore) save(key string, data []byte) error {
	locker, _ := s.kv.NewLock(s.ctx, s.getLockKey(key), nil)
	_, err := locker.Lock(s.ctx)
	if err != nil {
		return err
	}
	defer locker.Unlock(s.ctx)

	err = s.kv.Put(s.ctx, key, data, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *ValkeyrieStore) get(key string) ([]byte, error) {
	locker, _ := s.kv.NewLock(s.ctx, s.getLockKey(key), nil)
	_, err := locker.Lock(s.ctx)
	if err != nil {
		return nil, err
	}
	defer locker.Unlock(s.ctx)

	exists, err := s.kv.Exists(s.ctx, key, nil)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	pair, err := s.kv.Get(s.ctx, key, nil)
	if err != nil {
		return nil, err
	}

	return pair.Value, nil
}

// GetAccount returns ACME Account.
func (s *ValkeyrieStore) GetAccount(resolverName string) (*Account, error) {
	key := s.getKey(resolverName, "account")
	data, err := s.get(key)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	var account Account

	err = json.Unmarshal(data, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (s *ValkeyrieStore) getKey(resolverName string, keyType string) string {
	return resolverName + "_" + keyType
}

func (s *ValkeyrieStore) getLockKey(key string) string {
	return key + "_lock"
}

// SaveAccount stores ACME Account.
func (s *ValkeyrieStore) SaveAccount(resolverName string, account *Account) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}
	key := s.getKey(resolverName, "account")
	err = s.save(key, data)
	if err != nil {
		return err
	}
	return nil
}

// GetCertificates returns ACME Certificates list.
func (s *ValkeyrieStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	key := s.getKey(resolverName, "certificates")
	data, err := s.get(key)
	if err != nil {
		return nil, err
	}

	var certificates []*CertAndStore

	if data == nil {
		return certificates, nil
	}

	err = json.Unmarshal(data, &certificates)
	if err != nil {
		return nil, err
	}

	return certificates, nil
}

// SaveCertificates stores ACME Certificates list.
func (s *ValkeyrieStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	data, err := json.Marshal(certificates)
	if err != nil {
		return err
	}
	key := s.getKey(resolverName, "certificates")
	err = s.save(key, data)
	if err != nil {
		return err
	}

	return nil
}

func (s *ValkeyrieStore) WatchCertificateChanges(resolverName string) (<-chan *store.KVPair, error) {
	key := s.getKey(resolverName, "certificates")
	locker, _ := s.kv.NewLock(s.ctx, s.getLockKey(key), nil)
	_, err := locker.Lock(s.ctx)
	if err != nil {
		return nil, err
	}
	defer locker.Unlock(s.ctx)
	return s.kv.Watch(s.ctx, key, nil)
}
