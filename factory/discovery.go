package factory

import (
	"fmt"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/discovery/impl/etcd"
	"github.com/995933447/microgosuit/discovery/impl/filecachedproxy"
	"github.com/995933447/microgosuit/env"
	"github.com/etcd-io/etcd/clientv3"
	"sync"
	"time"
)

var (
	discover        discovery.Discovery
	initDiscoveryMu sync.RWMutex
)

func NewSpecDiscovery(discoveryName string) (discovery.Discovery, error) {
	switch discoveryName {
	case env.DiscoveryEtcd:
		return etcd.NewDiscovery(time.Second*5, clientv3.Config{
			Endpoints:   env.MustMeta().Etcd.Endpoints,
			DialTimeout: time.Duration(env.MustMeta().Etcd.ConnectTimeoutMs) * time.Millisecond,
		})
	}
	return nil, fmt.Errorf("no support discovery type(%s)", discoveryName)
}

func GetOrMakeDiscovery() (discovery.Discovery, error) {
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

	makeNotProxyDiscovery := func(discoveryName string) (discovery.Discovery, error) {
		switch discoveryName {
		case env.DiscoveryEtcd:
			return etcd.NewDiscovery(time.Second*5, clientv3.Config{
				Endpoints:   env.MustMeta().Etcd.Endpoints,
				DialTimeout: time.Duration(env.MustMeta().Etcd.ConnectTimeoutMs) * time.Millisecond,
			})
		}
		return nil, fmt.Errorf("no support discovery type(%s)", discoveryName)
	}

	var err error
	if env.MustMeta().Discovery != env.DiscoveryFileCacheProxy {
		discover, err = makeNotProxyDiscovery(env.MustMeta().Discovery)
		if err != nil {
			return nil, err
		}
		return discover, nil
	}

	conn, err := makeNotProxyDiscovery(env.MustMeta().DiscoveryProxy.Conn)
	if err != nil {
		return nil, err
	}

	discover = filecachedproxy.NewDiscovery(env.MustMeta().DiscoveryProxy.Dir, conn)

	return discover, nil
}
