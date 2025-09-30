package skeleton

import (
	"log"
	"os"
	"time"

	"github.com/995933447/confloader"
	"github.com/995933447/runtimeutil"
)

type ProtocGenConf struct {
	ProjectDir        string `json:"project_dir"`
	DirNamingMethod   string `json:"dir_naming_method"`
	GrpcResolveSchema string `json:"grpc_resolve_schema"`
	DiscoverPrefix    string `json:"discover_prefix"`
	EnabledHealth     bool   `json:"enabled_health"`
	Debug             bool   `json:"debug"`
}

var (
	protocGenConf       ProtocGenConf
	loadedProtocGenConf bool
)

const DefaultGrpcResolveSchema = "microgosuit"
const DefaultDiscoverPrefix = "microgosuit/"

func LoadProtocGenConf() error {
	confFilePath := os.Getenv(EnvKeyProtocGenConfFilePath)
	if confFilePath == "" {
		confFilePath = defaultProtocGenCfgFilePath
	}
	if _, err := os.Stat(confFilePath); err != nil {
		if !os.IsNotExist(err) {
			log.Println(runtimeutil.NewStackErr(err))
			return err
		}

		loadedProtocGenConf = true
		protocGenConf.GrpcResolveSchema = DefaultGrpcResolveSchema
		protocGenConf.DiscoverPrefix = DefaultDiscoverPrefix
		return nil
	}
	err := confloader.NewLoader(confFilePath, time.Second, &protocGenConf).Load()
	if err != nil {
		return err
	}
	if protocGenConf.GrpcResolveSchema == "" {
		protocGenConf.GrpcResolveSchema = DefaultGrpcResolveSchema
		protocGenConf.DiscoverPrefix = DefaultDiscoverPrefix
	}
	loadedProtocGenConf = true
	return nil
}

func MustGetProtocGenConf() *ProtocGenConf {
	if !loadedProtocGenConf {
		panic("protoc-gen conf not loaded")
	}
	return &protocGenConf
}
