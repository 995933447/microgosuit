package discovery

const (
	FileCachedProxyProtoFileExt = ".json"
)

// FileCachedProxyContract file cache proxy system is designed to cache remote discovery by used local file system.
// Provide capacity of accessing handler config in high performance and alleviate workload of remote discovery and
// also get HA(local file could be instead of remote discovery in temporary while remote discovery died).
// FileCachedProxyContract use to have a suite of contract designing your file cache proxy.
type FileCachedProxyContract struct {
}

func (*FileCachedProxyContract) GetCacheFilePathBySrv(dir, srvName string) string {
	return dir + "/" + srvName + FileCachedProxyProtoFileExt
}
