package acme

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	// github.com/gusaul/go-dynamock
	//"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/dynamodb"
	//"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

func TestDynamoStore_GetAccount(t *testing.T) {
	// Needs mock database,
	// what's the best way to add one?
}

func TestDynamoStore_SaveAccount(t *testing.T) {
	redisURL := "TODO: URL"

	s := NewDynamoStore(redisURL)

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
