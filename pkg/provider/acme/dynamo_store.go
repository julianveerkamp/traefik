package acme

// import (
//
//	"context"
//	"github.com/go-redis/redis/v8"
//
// )

var _ Store = (*DynamoStore)(nil)

// DynamoStore Stores implementation DynamoDB database.
type DynamoStore struct {
	Addr string
}

// NewDynamoStore initializes a new DynamoStore with an URL.
func NewDynamoStore(Addr string) *DynamoStore {
	store := &DynamoStore{Addr: Addr}

	return store
}

// GetAccount returns ACME Account.
func (s *DynamoStore) GetAccount(resolverName string) (*Account, error) {
	panic("TODO")

	return nil, nil
}

// SaveAccount stores ACME Account.
func (s *DynamoStore) SaveAccount(resolverName string, account *Account) error {
	panic("TODO")

	return nil
}

// GetCertificates returns ACME Certificates list.
func (s *DynamoStore) GetCertificates(resolverName string) ([]*CertAndStore, error) {
	panic("TODO")

	return nil, nil
}

// SaveCertificates stores ACME Certificates list.
func (s *DynamoStore) SaveCertificates(resolverName string, certificates []*CertAndStore) error {
	panic("TODO")

	return nil
}
