//go:build !windows
// +build !windows

package log

const (
	defaultFastlogCfgFilePath = "/etc/microgosuit/log/fastlog.json"
	defaultFastlogLogBaseDir  = "/data/micrigosuit/log/"
)
