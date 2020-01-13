// +build !dev

package main

import "go.uber.org/zap"

const (
	isDev              = false
	defaultZapEncoding = "json"
	defaultZapLevel    = zap.InfoLevel
)
