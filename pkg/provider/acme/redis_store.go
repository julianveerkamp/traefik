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

// GetAccount returns ACME Account.
func (s *RedisStore) GetAccount(resolverName string) (*Account, error) {
	var account Account
	//if err := s.rdb.HGetAll(s.ctx, resolverName).Scan(&account); err != nil {
	//	return nil, err
	//}
	//accountJSON, err := s.rdb.Bytes(s.rh.JSONGet("student", "."))
	res, err := s.rh.JSONGet(resolverName, ".")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(res.([]byte), &account)
	if err != nil {
		log.Fatalf("Failed to JSON Unmarshal")
		return nil, err
	}

	return &account, nil
}

// SaveAccount stores ACME Account.
func (s *RedisStore) SaveAccount(resolverName string, account *Account) error {
	panic("TODO")

	return nil
}

// GetCertificates returns ACME Certificates list.
func (s *RedisStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	panic("TODO")

	return nil, nil
}

// SaveCertificates stores ACME Certificates list.
func (s *RedisStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	panic("TODO")

	return nil
}
