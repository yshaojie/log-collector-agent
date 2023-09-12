package utils

import "os"

func IsDebugEnv() bool {
	return len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0
}
