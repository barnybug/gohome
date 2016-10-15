package ener314

import "log"

const (
	LOG_TRACE = iota
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

var logLevel = LOG_INFO

func SetLevel(level int) {
	logLevel = level
}

func logs(level int, msg ...interface{}) {
	if level >= logLevel {
		log.Println(msg...)
	}
}

func logf(level int, fmt string, msg ...interface{}) {
	if level >= logLevel {
		log.Printf(fmt, msg...)
	}
}
