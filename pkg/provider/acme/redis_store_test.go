package acme

import (
	"github.com/go-redis/redis/v8"
	"github.com/nitishm/go-rejson/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisStore_GetAccount(t *testing.T) {
	addr := "localhost:6379"
	rh := rejson.NewReJSONHandler()
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	rh.SetGoRedisClient(rdb)

	account := Account{
		Email: "some42@email.com",
	}

	_, exp_err := rh.JSONSet("test", ".", account)
	require.NoError(t, exp_err)

	s := NewRedisStore(addr)

	res, err := s.GetAccount("test")

	require.NoError(t, err)
	assert.Equal(t, account.Email, res.Email)
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
