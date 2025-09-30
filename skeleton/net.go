package skeleton

import (
	"github.com/995933447/gonetutil"
)

func RandomAvailableRpcPort() (int, error) {
	port := 21000
	for {
		ok, err := gonetutil.IsPortAvailable(port)
		if err != nil {
			return 0, err
		}

		if ok {
			break
		}

		port++
	}

	return port, nil
}
