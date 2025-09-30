package log

import (
	"github.com/995933447/fastlog"
	"github.com/995933447/fastlog/logger"
)

func InitFastlog(moduleName string, cfgFilePath string) error {
	fastlog.SetModuleName(moduleName)

	if cfgFilePath == "" {
		cfgFilePath = defaultFastlogCfgFilePath
	}

	if err := fastlog.InitDefaultCfgLoader(cfgFilePath, &logger.LogConf{
		File: logger.FileLogConf{
			LogInfoBeforeFileSizeBytes:  -1,
			LogDebugBeforeFileSizeBytes: -1,
			Level:                       logger.LevelToStrMap[logger.LevelDebug],
			DefaultLogDir:               defaultFastlogLogBaseDir + moduleName + "/log",
			BillLogDir:                  defaultFastlogLogBaseDir + moduleName + "/bill",
			StatLogDir:                  defaultFastlogLogBaseDir + moduleName + "/stat",
		}, 
		AlertLevel: logger.LevelToStrMap[logger.LevelWarn],
	}); err != nil {
		return err
	}

	if err := fastlog.InitDefaultLogger(nil); err != nil {
		return err
	}

	return nil
}
