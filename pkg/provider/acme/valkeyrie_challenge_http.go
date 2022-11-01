package acme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-acme/lego/v4/challenge/http01"
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

	return c.kv.Put(c.ctx, c.getValkeyrieKey(token, domain), []byte(keyAuth), nil)
}

// CleanUp cleans the challenges when certificate is obtained.
func (c *ValkeyrieChallengeHTTP) CleanUp(domain, token, _ string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.kv == nil {
		return nil
	}

	valkeyrieKey := c.getValkeyrieKey(token, domain)

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

	token, err := c.getPathParam(req.URL)
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

func (*ValkeyrieChallengeHTTP) getPathParam(uri *url.URL) (string, error) {
	exp := regexp.MustCompile(fmt.Sprintf(`^%s([^/]+)/?$`, http01.ChallengePath("")))
	parts := exp.FindStringSubmatch(uri.Path)

	if len(parts) != 2 {
		return "", errors.New("missing token")
	}

	return parts[1], nil
}

func (*ValkeyrieChallengeHTTP) getValkeyrieKey(token, domain string) string {
	return domain + "_" + token
}
