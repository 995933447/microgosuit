package proxy

import (
	"context"
	"encoding/json"
	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/env"
	"github.com/995933447/microgosuit/factory"
	"github.com/995933447/microgosuit/log"
	"github.com/995933447/reflectutil"
	"github.com/coreos/etcd/pkg/ioutil"
	"golang.org/x/sync/errgroup"
	"os"
	"sync"
	"time"
)

func NewProxy() (*Proxy, error) {
	discover, err := factory.NewSpecDiscovery(env.MustMeta().DiscoveryProxy.Conn)
	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		dir:      env.MustMeta().DiscoveryProxy.Dir,
		discover: discover,
	}

	go func() {
		watchCfg(proxy)
	}()

	return proxy, nil
}

func watchCfg(proxy *Proxy) {
	var (
		oldCfg     env.DiscoveryProxy
		oldConnCfg interface{}
	)
	switch env.MustMeta().DiscoveryProxy.Conn {
	case env.DiscoveryEtcd:
		oldConnCfg = env.MustMeta().Etcd
	}
	if err := reflectutil.CopySameFields(env.MustMeta().DiscoveryProxy, &oldCfg); err != nil {
		log.Logger.Error(nil, err)
	}

	go func() {
		refreshCfgTk := time.NewTicker(3 * time.Second)
		defer refreshCfgTk.Stop()
		for {
			if proxy.isExited {
				break
			}

			<-refreshCfgTk.C

			cfg := env.MustMeta().DiscoveryProxy
			if cfg.Conn == oldCfg.Conn && cfg.Dir == oldCfg.Dir {
				var isDiff bool

				switch cfg.Conn {
				case env.DiscoveryEtcd:
					connCfg := env.MustMeta().Etcd
					if connCfg.ConnectTimeoutMs != oldConnCfg.(env.Etcd).ConnectTimeoutMs {
						isDiff = true
						break
					}

					if len(connCfg.Endpoints) != len(oldConnCfg.(env.Etcd).Endpoints) {
						isDiff = true
						break
					}

					for _, endpoint := range connCfg.Endpoints {
						var existed bool

						for _, oldEndpoint := range oldConnCfg.(env.Etcd).Endpoints {
							if endpoint != oldEndpoint {
								continue
							}
							existed = true
							break
						}

						if !existed {
							isDiff = true
							oldConnCfg = connCfg
							break
						}
					}
				default:
					log.Logger.Error(nil, "no support discovery type:"+cfg.Conn)
				}

				if !isDiff {
					continue
				}
			}

			discover, err := factory.GetOrMakeDiscovery()
			if err != nil {
				log.Logger.Error(nil, err)
				continue
			}
			proxy.discover = discover
			proxy.dir = cfg.Dir

			proxy.mu.RLock()
			if proxy.isExited {
				proxy.mu.RUnlock()
				break
			}
			proxy.rerun()
			proxy.mu.RUnlock()

			oldCfg = env.DiscoveryProxy{}
			if err = reflectutil.CopySameFields(cfg, &oldCfg); err != nil {
				log.Logger.Error(nil, err)
			}
		}
	}()
}

type Proxy struct {
	dir      string
	discover discovery.Discovery

	mu                sync.RWMutex
	isExited          bool
	stopOrRerunSignCh chan struct{}
	exitSignCh        chan struct{}
	discovery.FileCachedProxyContract
}

func (p *Proxy) rerun() {
	p.stopOrRerunSignCh <- struct{}{}
}

func (p *Proxy) Run() error {
	p.discover.OnSrvUpdated(func(ctx context.Context, evt discovery.Evt, srv *discovery.Service) {
		switch evt {
		case discovery.EvtUpdated:
			if err := p.SyncSrvToLocalFile(srv); err != nil {
				log.Logger.Error(nil, err)
			}
		case discovery.EvtDeleted:
			err := os.Remove(p.getSrvFilePath(srv.SrvName))
			if err != nil {
				log.Logger.Error(nil, err)
			}
		}
	})

	var eg errgroup.Group

	eg.Go(func() error {
		for {
			p.mu.RLock()
			if p.isExited {
				p.mu.RUnlock()
				break
			}
			p.mu.RUnlock()

			if _, err := p.discover.LoadAll(context.Background()); err != nil {
				return err
			}

			<-p.stopOrRerunSignCh
		}
		return nil
	})

	eg.Go(func() error {
		<-p.exitSignCh
		p.mu.Lock()
		p.isExited = true
		p.stopOrRerunSignCh <- struct{}{}
		p.mu.Unlock()
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func (p *Proxy) SyncSrvToLocalFile(srv *discovery.Service) error {
	path := p.FileCachedProxyContract.GetCacheFilePathBySrv(p.dir, srv.SrvName)
	srvJson, err := json.Marshal(srv)
	if err != nil {
		return err
	}
	err = ioutil.WriteAndSyncFile(path, srvJson, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (p *Proxy) getSrvFilePath(srvName string) string {
	return p.dir + "/" + srvName + ".json"
}
