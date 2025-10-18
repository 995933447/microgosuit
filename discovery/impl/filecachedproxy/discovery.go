package filecachedproxy

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/995933447/microgosuit/discovery"
	"github.com/995933447/microgosuit/discovery/util"
	"github.com/995933447/microgosuit/log"
	"github.com/howeyc/fsnotify"
)

type Discovery struct {
	discovery.FileCachedProxyContract
	*util.SpecSrvMuFactory

	dir             string
	conn            discovery.Discovery
	mu              sync.RWMutex
	srvMap          map[string]*discovery.Service
	onSrvUpdate     discovery.OnSrvUpdatedFunc
	unwatchSrvChMap map[string]chan struct{}
	isWatched       bool
	watchedServices map[string]struct{}
}

func (d *Discovery) LoadAll(_ context.Context) ([]*discovery.Service, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	files, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}

	var services []*discovery.Service
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if !strings.HasSuffix(fileName, ".json") {
			continue
		}

		suffixPos := strings.IndexByte(fileName, '.')
		if suffixPos <= 0 {
			continue
		}

		srvName := fileName[:suffixPos]

		srv, err := d.getSrvFromLocalFile(srvName)
		if err != nil {
			return nil, err
		}

		if _, ok := d.watchedServices[srvName]; !ok {
			go func() {
				if err = d.watchOne(srvName); err != nil {
					log.Logger.Error(nil, err)
					return
				}
			}()
		}

		d.srvMap[srvName] = srv
		services = append(services, srv)
	}

	return services, nil
}

func (d *Discovery) Register(ctx context.Context, srvName string, node *discovery.Node) error {
	return d.conn.Register(ctx, srvName, node)
}

func (d *Discovery) Unregister(ctx context.Context, srvName string, node *discovery.Node, remove bool) error {
	return d.conn.Unregister(ctx, srvName, node, remove)
}

func (d *Discovery) UnregisterAll(ctx context.Context, srvName string) error {
	return d.conn.UnregisterAll(ctx, srvName)
}

func (d *Discovery) getSrvFilePath(srvName string) string {
	return d.FileCachedProxyContract.GetCacheFilePathBySrv(d.dir, srvName)
}

func (d *Discovery) getSrvFromLocalFile(srvName string) (*discovery.Service, error) {
	filePath := d.getSrvFilePath(srvName)
	fp, err := os.Open(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return nil, discovery.ErrSrvNotFound
	}
	defer fp.Close()
	buf, err := io.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	var srv discovery.Service
	if len(buf) == 0 {
		return nil, discovery.ErrSrvNotFound
	}
	err = json.Unmarshal(buf, &srv)
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (d *Discovery) Discover(ctx context.Context, srvName string) (*discovery.Service, error) {
	d.mu.RLock()
	srv, ok := d.srvMap[srvName]
	d.mu.RUnlock()
	if ok {
		return srv, nil
	}

	d.mu.Lock()
	oneSrvMu := d.MakeOrGetOpOneSrvMu(srvName)
	d.mu.Unlock()

	oneSrvMu.Lock()
	defer oneSrvMu.Unlock()

	srv, ok = d.srvMap[srvName]
	if ok {
		return srv, nil
	}

	if _, ok = d.watchedServices[srvName]; !ok {
		go func() {
			if err := d.watchOne(srvName); err != nil {
				log.Logger.Error(nil, err)
				return
			}
		}()
	}

	srv, err := d.getSrvFromLocalFile(srvName)
	if err != nil {
		if err != discovery.ErrSrvNotFound {
			return nil, err
		}
		srv, err = d.conn.Discover(ctx, srvName)
		if err != nil {
			return nil, err
		}
	} else {
		d.srvMap[srvName] = srv
	}

	if d.onSrvUpdate != nil {
		d.onSrvUpdate(ctx, discovery.EvtUpdated, srv)
	}

	return srv, nil
}

func (d *Discovery) watchOne(srvName string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	d.mu.Lock()
	oneSrvMu := d.MakeOrGetOpOneSrvMu(srvName)
	d.mu.Unlock()

	oneSrvMu.RLock()
	if _, ok := d.watchedServices[srvName]; ok {
		oneSrvMu.RUnlock()
		return nil
	}
	oneSrvMu.RUnlock()

	oneSrvMu.Lock()
	if _, ok := d.watchedServices[srvName]; ok {
		oneSrvMu.Unlock()
		return nil
	}
	d.watchedServices[srvName] = struct{}{}
	oneSrvMu.Unlock()

	filePath := d.getSrvFilePath(srvName)
	var (
		unwatchCh    = make(chan struct{})
		watchSuccess bool
	)
	for {
		if !watchSuccess {
			err = watcher.Watch(filePath)
			if err != nil {
				log.Logger.Error(nil, err)
				time.Sleep(5 * time.Second)
				continue
			}

			watchSuccess = true

			d.unwatchSrvChMap[srvName] = unwatchCh

			oneSrvMu = d.MakeOrGetOpOneSrvMu(srvName)
			oneSrvMu.Lock()

			srv, err := d.getSrvFromLocalFile(srvName)
			if err != nil {
				delete(d.srvMap, srvName)
				oneSrvMu.Unlock()

				log.Logger.Error(nil, err)
				continue
			}

			d.srvMap[srvName] = srv
			oneSrvMu.Unlock()
		}

		select {
		case <-unwatchCh:
			goto out
		case evt := <-watcher.Event:
			if evt.IsDelete() || evt.IsRename() {
				d.mu.Lock()
				oneSrvMu = d.MakeOrGetOpOneSrvMu(srvName)
				d.mu.Unlock()

				oneSrvMu.Lock()
				delete(d.srvMap, srvName)
				oneSrvMu.Unlock()

				if d.onSrvUpdate != nil {
					d.onSrvUpdate(context.Background(), discovery.EvtDeleted, &discovery.Service{
						SrvName: srvName,
					})
				}
			}

			if evt.IsAttrib() {
				continue
			}

			d.mu.Lock()
			oneSrvMu = d.MakeOrGetOpOneSrvMu(srvName)
			d.mu.Unlock()

			oneSrvMu.Lock()
			srv, err := d.getSrvFromLocalFile(srvName)
			if err != nil {
				delete(d.srvMap, srvName)
				oneSrvMu.Unlock()

				log.Logger.Error(nil, err)
				continue
			}
			d.srvMap[srvName] = srv
			oneSrvMu.Unlock()

			if d.onSrvUpdate != nil {
				d.onSrvUpdate(context.Background(), discovery.EvtUpdated, srv)
			}
		}
	}
out:
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.watchedServices[srvName]; ok {
		return nil
	}
	delete(d.unwatchSrvChMap, srvName)
	delete(d.watchedServices, srvName)
	return nil
}

func (d *Discovery) OnSrvUpdated(fn discovery.OnSrvUpdatedFunc) {
	d.onSrvUpdate = fn
}

func (d *Discovery) Unwatch() {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, unwatchCh := range d.unwatchSrvChMap {
		unwatchCh <- struct{}{}
	}
}

var _ discovery.Discovery = (*Discovery)(nil)

func NewDiscovery(dir string, conn discovery.Discovery) discovery.Discovery {
	discover := &Discovery{
		dir:             dir,
		conn:            conn,
		srvMap:          map[string]*discovery.Service{},
		unwatchSrvChMap: map[string]chan struct{}{},
		watchedServices: map[string]struct{}{},
	}
	discover.SpecSrvMuFactory = util.NewSpecSrvMuFactory()
	return discover
}
