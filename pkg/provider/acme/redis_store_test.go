package acme

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRedisStore_GetAccount(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	ctx := context.Background()

	account := Account{
		Email: "some42@email.com",
	}

	if _, err := rdb.Pipelined(ctx, func(rdb redis.Pipeliner) error {
		rdb.HSet(ctx, "test", "email", account.Email)
		return nil
	}); err != nil {
		panic(err)
	}

	s := NewRedisStore(server.Addr())

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
