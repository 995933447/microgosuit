package factory

import (
	"fmt"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/discovery/impl/etcd"
	"github.com/995933447/microgosuit/discovery/impl/filecachedproxy"
	"github.com/995933447/microgosuit/env"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

var (
	discover        discovery.Discovery
	initDiscoveryMu sync.RWMutex
)

var CustomMakeDiscoveryFunc func(discoveryName string) (discovery.Discovery, error)

func NewSpecDiscovery(discoverKeyPrefix, discoveryName string) (discovery.Discovery, error) {
	switch discoveryName {
	case env.DiscoveryEtcd:
		return etcd.NewDiscovery(discoverKeyPrefix, time.Second*5, clientv3.Config{
			Endpoints:   env.MustMeta().Etcd.Endpoints,
			DialTimeout: time.Duration(env.MustMeta().Etcd.ConnectTimeoutMs) * time.Millisecond,
		})
	default:
		if CustomMakeDiscoveryFunc != nil {
			return CustomMakeDiscoveryFunc(discoveryName)
		}
	}
	return nil, fmt.Errorf("no support discovery type(%s)", discoveryName)
}

func GetOrMakeDiscovery(discoverKeyPrefix string) (discovery.Discovery, error) {
	initDiscoveryMu.RLock()
	if discover != nil {
		initDiscoveryMu.RUnlock()
		return discover, nil
	}
	initDiscoveryMu.RUnlock()

	initDiscoveryMu.Lock()
	defer initDiscoveryMu.Unlock()

	if discover != nil {
		return discover, nil
	}

	var err error
	if env.MustMeta().Discovery != env.DiscoveryFileCacheProxy {
		discover, err = NewSpecDiscovery(discoverKeyPrefix, env.MustMeta().Discovery)
		if err != nil {
			return nil, err
		}
		return discover, nil
	}

	conn, err := NewSpecDiscovery(discoverKeyPrefix, env.MustMeta().DiscoveryProxy.Conn)
	if err != nil {
		return nil, err
	}

	discover = filecachedproxy.NewDiscovery(env.MustMeta().DiscoveryProxy.Dir, conn)

	return discover, nil
}
