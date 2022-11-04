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
	"github.com/traefik/traefik/v2/pkg/log"
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
	log.WithoutContext().Infoln("Created new ValkeyrieChallengeHTTP for " + storeName + " at " + addr)
	logger := log.WithoutContext().WithField(log.ProviderName, "acme")

	ctx := context.Background()
	kv, err := valkeyrie.NewStore(ctx, storeName, []string{addr}, config)
	if err != nil {
		logger.Error(err)
	}

	return &ValkeyrieChallengeHTTP{
		ctx: ctx,
		kv:  kv,
	}
}

// Present presents a challenge to obtain new ACME certificate.
func (c *ValkeyrieChallengeHTTP) Present(domain, token, keyAuth string) error {
	log.WithoutContext().Infoln("Present for " + domain + " with token " + token + " and keyAuth " + keyAuth)
	c.lock.Lock()
	defer c.lock.Unlock()

	isMain, err := c.isChallengeMain()
	if err != nil {
		return fmt.Errorf("error while checking if instance is the HTTP challenge main: %s", err)
	}

	if !isMain {
		// TODO: return error here or return nil (which means that the check for the token will fail)
		return fmt.Errorf("instance is not HTTP Challenge main. It is not allowed to present new challenges")
	}

	// Present
	valkeyrieKey := c.getValkeyrieKey(token, domain)
	valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
	log.WithoutContext().Infoln("New lock... " + valkeyrieKeyLock)
	locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, &store.LockOptions{TTL: 20 * time.Second, Value: []byte("asfd")})
	log.WithoutContext().Infoln("Trying to get lock " + valkeyrieKeyLock)
	_, err = locker.Lock(c.ctx)
	if err != nil {
		log.WithoutContext().Infoln("Lock error: " + err.Error())
	}
	defer logUnlocking(c.ctx, locker, valkeyrieKeyLock)
	log.WithoutContext().Infoln("Got lock " + valkeyrieKeyLock)

	err = c.kv.Put(c.ctx, valkeyrieKey, []byte(keyAuth), nil)
	log.WithoutContext().Infoln("Put value into kv")
	return err
}

func logUnlocking(ctx context.Context, locker store.Locker, lockName string) {
	log.WithoutContext().Infoln("Trying to unlock lock " + lockName)
	err := locker.Unlock(ctx)
	if err != nil {
		log.WithoutContext().Infoln("Unlock error: " + err.Error())
	}
	log.WithoutContext().Infoln("Unlocked lock " + lockName)
}

// returns true if the current instance already is the main or got "elected" as the main by being the first
// instance which wants to become the main because there was no other main before or after the old main's TTL
// expired
// returns false if another instance is the main
func (c *ValkeyrieChallengeHTTP) isChallengeMain() (bool, error) {
	// Get lock for main
	mainKey := "http_challenge_main"
	mainLockKey := "http_challenge_main_lock"

	locker, _ := c.kv.NewLock(c.ctx, mainLockKey, nil)
	log.WithoutContext().Infoln("Trying to get lock " + mainLockKey)
	_, err := locker.Lock(c.ctx)
	if err != nil {
		log.WithoutContext().Infoln("Lock error: " + err.Error())
	}
	defer logUnlocking(c.ctx, locker, mainLockKey)
	log.WithoutContext().Infoln("Got lock " + mainLockKey)

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
	log.WithoutContext().Infoln("CleanUp for " + domain + " with token " + token)

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.kv == nil {
		return nil
	}

	valkeyrieKey := c.getValkeyrieKey(token, domain)
	valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
	log.WithoutContext().Infoln("New lock... " + valkeyrieKeyLock)
	locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, nil)
	log.WithoutContext().Infoln("Trying to get lock " + valkeyrieKeyLock)
	locker.Lock(c.ctx)
	defer locker.Unlock(c.ctx)

	log.WithoutContext().Infoln("Got lock " + valkeyrieKeyLock)

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
	log.WithoutContext().Infoln("Timeout")
	return 60 * time.Second, 5 * time.Second
}

func (c *ValkeyrieChallengeHTTP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.WithoutContext().Infoln("Serving HTTP for " + req.RequestURI)
	ctx := log.With(req.Context(), log.Str(log.ProviderName, "acme"))
	logger := log.FromContext(ctx)

	token, err := getPathParam(req.URL)
	if err != nil {
		logger.Errorf("Unable to get token: %v.", err)
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if token != "" {
		domain, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			logger.Debugf("Unable to split host and port: %v. Fallback to request host.", err)
			domain = req.Host
		}

		tokenValue := c.getTokenValue(ctx, token, domain)
		if len(tokenValue) > 0 {
			rw.WriteHeader(http.StatusOK)
			_, err = rw.Write(tokenValue)
			if err != nil {
				logger.Errorf("Unable to write token: %v", err)
			}
			return
		}
	}

	rw.WriteHeader(http.StatusNotFound)
}

func (c *ValkeyrieChallengeHTTP) getTokenValue(ctx context.Context, token, domain string) []byte {
	logger := log.FromContext(ctx)
	logger.Debugf("Retrieving the ACME challenge for %s (token %q)...", domain, token)

	var result []byte

	operation := func() error {
		c.lock.RLock()
		defer c.lock.RUnlock()

		valkeyrieKey := c.getValkeyrieKey(token, domain)
		valkeyrieKeyLock := c.getValkeyrieKeyLock(token, domain)
		log.WithoutContext().Infoln("New lock... " + valkeyrieKeyLock)
		locker, _ := c.kv.NewLock(c.ctx, valkeyrieKeyLock, nil)
		log.WithoutContext().Infoln("Trying to get lock " + valkeyrieKeyLock)
		locker.Lock(c.ctx)
		defer locker.Unlock(c.ctx)

		log.WithoutContext().Infoln("Got lock " + valkeyrieKeyLock)

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
		logger.Errorf("Error getting challenge for token retrying in %s", time)
	}

	ebo := backoff.NewExponentialBackOff()
	ebo.MaxElapsedTime = 60 * time.Second
	err := backoff.RetryNotify(safe.OperationWithRecover(operation), ebo, notify)
	if err != nil {
		logger.Errorf("Cannot retrieve the ACME challenge for %s (token %q): %v", domain, token, err)
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
