package acme

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisStore_GetAccount(t *testing.T) {
	// Needs mock database,
	// what's the best way to add one?
}

func TestRedisStore_SaveAccount(t *testing.T) {
	redisURL := "TODO: URL"

	s := NewRedisStore(redisURL)

	email := "some@email.com"

	err := s.SaveAccount("test", &Account{Email: email})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_ = `{
  "test": {
    "Account": {
      "Email": "some@email.com",
      "Registration": null,
      "PrivateKey": null,
      "KeyType": ""
    },
    "Certificates": null
  }
}`
	// Assert that  redis correctly save the account
	// assert.Equal(t, expected, string(file))
}
