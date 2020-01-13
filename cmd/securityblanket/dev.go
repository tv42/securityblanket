// +build dev

package main

import "go.uber.org/zap"

const (
	isDev              = true
	defaultZapEncoding = "console"
	defaultZapLevel    = zap.DebugLevel
)
