package util

import (
	"os"
	"strings"
)

func ExpandUser(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		path = strings.Replace(path, "~", os.Getenv("HOME"), 1)
	}
	return path
}
