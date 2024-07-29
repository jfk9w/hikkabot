package me3x

import "github.com/jfk9w-go/flu/logf"

const rootLoggerName = "me3x"

func log() logf.Interface {
	return logf.Get(rootLoggerName)
}
