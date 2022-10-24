package acme

import (
	"encoding/json"
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
	storedData := StoredData{
		Account:      &account,
		Certificates: nil,
	}

	_, exp_err := rh.JSONSet("test", ".", storedData)
	require.NoError(t, exp_err)

	s := NewRedisStore(addr)

	res, err := s.GetAccount("test")

	require.NoError(t, err)
	assert.Equal(t, account.Email, res.Email)
}

func TestRedisStore_SaveAccount(t *testing.T) {
	addr := "localhost:6379"

	s := NewRedisStore(addr)

	account := Account{
		Email: "some42@email.com",
	}

	err := s.SaveAccount("test", &account)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	rh := rejson.NewReJSONHandler()
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	rh.SetGoRedisClient(rdb)

	res, actErr := rh.JSONGet("test", ".")
	require.NoError(t, actErr)

	var actual StoredData
	jsonErr := json.Unmarshal(res.([]byte), &actual)
	require.NoError(t, jsonErr)

	assert.Equal(t, &account, actual.Account)
}
