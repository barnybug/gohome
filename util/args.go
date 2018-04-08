package util

import (
	"strconv"
	"strings"
)

func KeywordArgs(args []string) map[string]string {
	ret := map[string]string{}
	for _, arg := range args {
		p := strings.SplitN(arg, "=", 2)
		if len(p) == 2 {
			ret[p[0]] = p[1]
		} else {
			ret[""] = p[0]
		}
	}
	return ret
}

func ParseArg(value string) interface{} {
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return num
	}
	return value
}

func ParseArgs(args []string) (string, map[string]interface{}) {
	kwargs := KeywordArgs(args)
	command := ""
	fields := map[string]interface{}{}
	for field, value := range kwargs {
		if field == "" {
			command = value
		} else {
			fields[field] = ParseArg(value)
		}
	}
	return command, fields
}
