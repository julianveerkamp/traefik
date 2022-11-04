package acme

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/julianveerkamp/redis"
	"github.com/kvtools/valkeyrie"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValkeyrieStore_EmptyGetAccount(t *testing.T) {
	mr := miniredis.RunT(t)

	s := NewValkeyrieStore(mr.Addr(), redis.StoreName, nil)

	account, err := s.GetAccount("test")
	require.NoError(t, err)

	assert.True(t, account == nil)
}

func TestValkeyrieStore_GetAccount(t *testing.T) {
	mr := miniredis.RunT(t)
	ctx := context.Background()
	addr := mr.Addr()
	kv, err := valkeyrie.NewStore(ctx, redis.StoreName, []string{addr}, nil)
	require.NoError(t, err)

	account := Account{
		Email: "some42@email.com",
	}

	data, err := json.Marshal(account)
	require.NoError(t, err)

	err = kv.Put(ctx, "test_account", data, nil)
	require.NoError(t, err)

	s := NewValkeyrieStore(addr, redis.StoreName, nil)

	actual, err := s.GetAccount("test")
	require.NoError(t, err)

	assert.Equal(t, &account, actual)
}

func TestValkeyrieStore_SaveAccount(t *testing.T) {
	mr := miniredis.RunT(t)
	s := NewValkeyrieStore(mr.Addr(), redis.StoreName, nil)

	account := Account{
		Email: "some42@email.com",
	}

	err := s.SaveAccount("test", &account)
	require.NoError(t, err)

	pair, err := s.kv.Get(s.ctx, "test_account", nil)
	require.NoError(t, err)

	var actual Account

	err = json.Unmarshal(pair.Value, &actual)
	require.NoError(t, err)

	assert.Equal(t, account, actual)
}
