package utils

import (
	"fmt"
	"os"
	"strconv"
)

func LookupEnvStr(name string, default_value string) string {
	v, ok := os.LookupEnv(name)
	if ok && v != "" {
		return v
	} else {
		return default_value
	}
}

func LookupEnvUint64(name string, default_value uint64) uint64 {
	v, ok := os.LookupEnv(name)
	if ok && v != "" {
		vi, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("parse %s with value %s to uint64 error: %v", name, v, err))
		}

		return uint64(vi)
	}

	return default_value
}
