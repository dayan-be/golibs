package etcd_util


import (
	"fmt"
	"sync"
	"time"

	"context"
	"go.etcd.io/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"github.com/dayan-be/gopkg/ttlcache"
)

// requestTimeout is the default value for query etcd server
const requestTimeout = 100 * time.Millisecond
const defaultExpiration = 1 * time.Minute

// cache will be shared for all keys
var cache *ttlcache.Cache = ttlcache.NewCache(defaultExpiration)

// defaultClient is created for easy use
var defaultEtcdClient *Client
var defaultEtcdClientGuard sync.RWMutex

func initEtcdClient() error {
	defaultEtcdClientGuard.RLock()
	if defaultEtcdClient != nil {
		defaultEtcdClientGuard.RUnlock()
		return nil
	}

	defaultEtcdClientGuard.RUnlock()
	if client, err := NewClient(); err == nil {
		defaultEtcdClientGuard.Lock()
		if defaultEtcdClient == nil {
			defaultEtcdClient = client
		}
		defaultEtcdClientGuard.Unlock()
		return nil
	} else {
		return err
	}
}

// Get using DefaultClient
func Get(key string, defaultValue string, opts ...clientv3.OpOption) (string, error) {
	return GetWithOptions(key, defaultValue, opts)
}

// GetWithOptions using DefaultClient
func GetWithOptions(key string, defaultValue string, opts *etcd.GetOptions) (string, error) {
	if err := initEtcdClient(); err != nil {
		return defaultValue, err
	}
	return defaultEtcdClient.GetWithCacheAndOpts(key, defaultValue, opts)
}

// Set using DefaultClient
func Set(key string, value string, opts ...clientv3.OpOption) (string, error) {
	return SetWithOptions(key, value, nil)
}

// SetWithOptions using DefaultClient
func SetWithOptions(key string, value string, opts *etcd.SetOptions) (string, error) {
	if err := initEtcdClient(); err != nil {
		return "", err
	}

	resp, err := defaultEtcdClient.Set(context.TODO(), key, value, opts)
	if err != nil || resp == nil {
		logrus.Error("Can't set key %s, %v", key, err)
		return "", err
	}

	cache.Set(key, value)
	return resp.Node.Value, nil
}

func (c *Client) GetWithCache(key string, defaultValue string) (string, error) {
	return c.GetWithCacheAndOpts(key, defaultValue, nil)
}

func (c *Client) GetWithCacheAndOpts(key string, defaultValue string, opts ...clientv3.OpOption) (string, error) {
	if len(key) == 0 {
		return defaultValue, fmt.Errorf("Can't get from empty key")
	}

	// Get from cache
	value, exists := cache.Get(key)
	if exists {
		// key in cache
		return value, nil
	}

	// key not in cache or expired
	//	logrus.Info("cache miss for key %s", key)
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := c.Get(ctx, key, opts...)
	value = defaultValue
	if err != nil {
		logrus.Error("Get from etcd error", err)
	} else if resp != nil && resp.Node != nil && len(resp.Node.Value) != 0 {
		value = resp.Node.Value
	}
	cache.Set(key, value)
	return value, err
}

