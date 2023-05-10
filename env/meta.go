package env

import (
	"github.com/995933447/confloader"
	"github.com/995933447/microgosuit/log"
	"sync"
	"time"
)

const (
	Dev  = "dev"
	Prod = "prod"
	Test = "test"
)

const (
	DiscoveryFileCacheProxy = "proxy"
	DiscoveryEtcd           = "etcd"
)

type Etcd struct {
	ConnectTimeoutMs int32    `json:"connect_timeout_ms"`
	Endpoints        []string `json:"endpoints"`
}

type DiscoveryProxy struct {
	Dir  string `json:"dir"`
	Conn string `json:"connection"`
}

type Meta struct {
	Env            string `json:"env"`
	Discovery      string `json:"discovery"`
	Etcd           `json:"etcd"`
	DiscoveryProxy `json:"discovery_proxy"`
}

func (m *Meta) IsDev() bool {
	return m.Env == Dev
}

func (m *Meta) IsTest() bool {
	return m.Env == Test
}

func (m *Meta) IsProd() bool {
	return m.Env == Prod
}

var (
	meta        *Meta
	hasInitMeta bool
	initMetaMu  sync.RWMutex
)

func InitMeta(cfgFilePath string) error {
	if cfgFilePath == "" {
		cfgFilePath = defaultCfgFilePath
	}

	if hasInitMeta {
		return nil
	}

	initMetaMu.Lock()
	defer initMetaMu.Unlock()

	if hasInitMeta {
		return nil
	}

	meta = &Meta{}
	cfgLoader := confloader.NewLoader(cfgFilePath, 5*time.Second, meta)
	if err := cfgLoader.Load(); err != nil {
		return err
	}

	hasInitMeta = true

	watchMetaErrCh := make(chan error)
	go cfgLoader.WatchToLoad(watchMetaErrCh)
	go func() {
		for {
			err := <-watchMetaErrCh
			if err != nil {
				log.Logger.Error(nil, err)
			}
		}
	}()

	return nil
}

func MustMeta() *Meta {
	if !hasInitMeta {
		panic("meta not init")
	}

	return meta
}
