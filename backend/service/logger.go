package service

import (
	"fmt"
	"os"
)

const (
	ColorRed   = "\033[31m"
	ColorReset = "\033[0m"
)

// LogError 输出红色错误日志到 stderr
func LogError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, ColorRed+format+ColorReset+"\n", args...)
}
