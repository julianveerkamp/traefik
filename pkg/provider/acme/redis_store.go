package acme

import (
	"context"
	"github.com/go-redis/redis/v8"
)

var _ Store = (*RedisStore)(nil)

// RedisStore Stores implementation redis database.
type RedisStore struct {
	ctx    context.Context
	client *redis.Client
}

// NewRedisStore initializes a new RedisStore with an URL.
func NewRedisStore(Addr string) *RedisStore {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: Addr,
	})
	store := &RedisStore{ctx: ctx, client: client}

	return store
}

// GetAccount returns ACME Account.
func (s *RedisStore) GetAccount(resolverName string) (*Account, error) {
	var account Account

	if err := s.client.HGetAll(s.ctx, resolverName).Scan(&account); err != nil {
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
