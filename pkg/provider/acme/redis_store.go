package acme

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/nitishm/go-rejson/v4"
	"log"
)

var _ Store = (*RedisStore)(nil)

// RedisStore Stores implementation redis database.
type RedisStore struct {
	ctx context.Context
	rdb *redis.Client
	rh  *rejson.Handler
}

// NewRedisStore initializes a new RedisStore with an URL.
func NewRedisStore(Addr string) *RedisStore {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: Addr,
	})
	rh := rejson.NewReJSONHandler()
	rh.SetGoRedisClient(client)
	store := &RedisStore{ctx: ctx, rdb: client, rh: rh}

	return store
}

func (s *RedisStore) save(resolverName string, storedData *StoredData) error {
	_, err := s.rh.JSONSet(resolverName, ".", storedData)
	if err != nil {
		return err
	}

	return nil
}

func (s *RedisStore) get(resolverName string) (*StoredData, error) {
	var data StoredData
	res, err := s.rh.JSONGet(resolverName, ".")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(res.([]byte), &data)
	if err != nil {
		log.Fatalf("Failed to JSON Unmarshal")
		return nil, err
	}

	return &data, nil
}

// GetAccount returns ACME Account.
func (s *RedisStore) GetAccount(resolverName string) (*Account, error) {
	data, err := s.get(resolverName)
	if err != nil {
		return nil, err
	}

	return data.Account, nil

}

// SaveAccount stores ACME Account.
func (s *RedisStore) SaveAccount(resolverName string, account *Account) error {
	storedData, err := s.get(resolverName)
	if err != nil {
		return err
	}

	storedData.Account = account
	save_err := s.save(resolverName, storedData)
	if save_err != nil {
		return save_err
	}

	return nil
}

// GetCertificates returns ACME Certificates list.
func (s *RedisStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	storedData, err := s.get(resolverName)
	if err != nil {
		return nil, err
	}

	return storedData.Certificates, nil
}

// SaveCertificates stores ACME Certificates list.
func (s *RedisStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	storedData, err := s.get(resolverName)
	if err != nil {
		return err
	}

	storedData.Certificates = certificates
	save_err := s.save(resolverName, storedData)
	if save_err != nil {
		return save_err
	}

	return nil
}
