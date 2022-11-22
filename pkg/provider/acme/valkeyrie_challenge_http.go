package acme

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/kvtools/valkeyrie"
	"github.com/kvtools/valkeyrie/store"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v2/pkg/logs"
	"github.com/traefik/traefik/v2/pkg/safe"
)

// ValkeyrieChallengeHTTP HTTP challenge provider implements challenge.Provider.
type ValkeyrieChallengeHTTP struct {
	lock sync.RWMutex

	kv  store.Store
	ctx context.Context
}

// NewChallengeHTTP creates a new ChallengeHTTP.
func NewValkeyrieChallengeHTTP(addr string, storeName string, config valkeyrie.Config) *ValkeyrieChallengeHTTP {
	logger := log.With().Str(logs.ProviderName, "acme").Logger()

	ctx := context.Background()
	kv, err := valkeyrie.NewStore(ctx, storeName, []string{addr}, config)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to create ValkeyrieChallengeHTTP")
	}

	return &ValkeyrieChallengeHTTP{
		ctx: ctx,
		kv:  kv,
	}
}

// Present presents a challenge to obtain new ACME certificate.
func (c *ValkeyrieChallengeHTTP) Present(domain, token, keyAuth string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	isMain, err := c.isChallengeMain()
	if err != nil {
		return fmt.Errorf("error while checking if instance is the HTTP challenge main: %s", err)
	}

	if !isMain {
		// TODO: return error here or return nil (which means that the check for the token will fail)
		return fmt.Errorf("instance is not HTTP challenge main. It is not allowed to present new challenges")
	}

	// Present
	valkeyrieKey := c.getValkeyrieKey(token, domain)
	valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
	locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, nil)
	_, err = locker.Lock(c.ctx)
	if err != nil {
		return err
	}
	defer locker.Unlock(c.ctx)

	return c.kv.Put(c.ctx, valkeyrieKey, []byte(keyAuth), nil)
}

// returns true if the current instance already is the main or got "elected" as the main by being the first
// instance which wants to become the main because there was no other main before or after the old main's TTL
// expired
// returns false if another instance is the main
func (c *ValkeyrieChallengeHTTP) isChallengeMain() (bool, error) {
	logger := log.With().Str(logs.ProviderName, "acme").Logger()
	// Get lock for main
	mainKey := "http_challenge_main"
	mainLockKey := "http_challenge_main_lock"

	locker, _ := c.kv.NewLock(c.ctx, mainLockKey, nil)
	_, err := locker.Lock(c.ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to Lock: " + mainLockKey)
		return false, err
	}
	defer locker.Unlock(c.ctx)

	hostname, err := os.Hostname()
	if err != nil {
		return false, err
	}

	// Check if main already exists
	// If the TTL is expired, this function should return false
	exists, err := c.kv.Exists(c.ctx, mainKey, nil)
	if err != nil {
		return false, err
	}

	if exists {
		// main exists -> Check if this instance already is main
		pair, err := c.kv.Get(c.ctx, mainKey, nil)
		if err != nil {
			return false, err
		}

		currentMainName := string(pair.Value)
		if hostname != currentMainName {
			return false, nil
		}
	}

	err = c.kv.Put(c.ctx, mainKey, []byte(hostname), &store.WriteOptions{TTL: 15 * time.Minute})

	if err != nil {
		return false, err
	}

	return true, nil
}

// CleanUp cleans the challenges when certificate is obtained.
func (c *ValkeyrieChallengeHTTP) CleanUp(domain, token, _ string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.kv == nil {
		return nil
	}

	valkeyrieKey := c.getValkeyrieKey(token, domain)
	valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
	locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, nil)
	locker.Lock(c.ctx)
	defer locker.Unlock(c.ctx)

	exists, err := c.kv.Exists(c.ctx, valkeyrieKey, nil)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	err = c.kv.Delete(c.ctx, valkeyrieKey)
	if err != nil {
		return err
	}

	return nil
}

// Timeout calculates the maximum of time allowed to resolved an ACME challenge.
func (c *ValkeyrieChallengeHTTP) Timeout() (timeout, interval time.Duration) {
	return 60 * time.Second, 5 * time.Second
}

func (c *ValkeyrieChallengeHTTP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := log.Ctx(req.Context()).With().Str(logs.ProviderName, "acme").Logger()

	token, err := getPathParam(req.URL)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to get token")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if token != "" {
		domain, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			logger.Debug().Err(err).Msg("Unable to split host and port. Fallback to request host.")
			domain = req.Host
		}

		tokenValue := c.getTokenValue(logger.WithContext(req.Context()), token, domain)
		if len(tokenValue) > 0 {
			rw.WriteHeader(http.StatusOK)
			_, err = rw.Write(tokenValue)
			if err != nil {
				logger.Error().Err(err).Msg("Unable to write token")
			}
			return
		}
	}

	rw.WriteHeader(http.StatusNotFound)
}

func (c *ValkeyrieChallengeHTTP) getTokenValue(ctx context.Context, token, domain string) []byte {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Retrieving the ACME challenge for %s (token %q)...", domain, token)

	var result []byte

	operation := func() error {
		c.lock.RLock()
		defer c.lock.RUnlock()

		valkeyrieKey := c.getValkeyrieKey(token, domain)
		valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
		locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, nil)
		locker.Lock(c.ctx)
		defer locker.Unlock(c.ctx)

		exists, err := c.kv.Exists(c.ctx, valkeyrieKey, nil)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("cannot find challenge for token %q (%s)", token, domain)
		}
		pair, err := c.kv.Get(c.ctx, valkeyrieKey, nil)
		if err != nil {
			return err
		}
		result = pair.Value

		return nil
	}

	notify := func(err error, time time.Duration) {
		logger.Error().Msgf("Error getting challenge for token retrying in %s", time)
	}

	ebo := backoff.NewExponentialBackOff()
	ebo.MaxElapsedTime = 60 * time.Second
	err := backoff.RetryNotify(safe.OperationWithRecover(operation), ebo, notify)
	if err != nil {
		logger.Error().Err(err).Msgf("Cannot retrieve the ACME challenge for %s (token %q)", domain, token)
		return []byte{}
	}

	return result
}

func (*ValkeyrieChallengeHTTP) getValkeyrieKey(token, domain string) string {
	return domain + "_" + token
}

func (*ValkeyrieChallengeHTTP) getValkeyrieKeyLock(token, domain string) string {
	return domain + "_" + token + "_lock"
}
