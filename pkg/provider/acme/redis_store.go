package acme

// import (
//
//	"context"
//	"github.com/go-redis/redis/v8"
//
// )

var _ Store = (*RedisStore)(nil)

// RedisStore Stores implementation redis database.
type RedisStore struct {
	url string
}

// NewRedisStore initializes a new RedisStore with an URL.
func NewRedisStore(url string) *RedisStore {
	store := &RedisStore{url: url}

	return store
}

// GetAccount returns ACME Account.
func (s *RedisStore) GetAccount(resolverName string) (*Account, error) {
	panic("TODO")

	return nil, nil
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
