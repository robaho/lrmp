package lrmp

import (
	"fmt"
	"io"
	"os"
)

var LogWriter = io.Writer(os.Stderr)

func logDebug(args ...interface{}) {
	if isDebug() {
		fmt.Fprintln(LogWriter, args)
	}
}
func isDebug() bool { return true }

func logError(args ...interface{}) {
	fmt.Fprintln(LogWriter, args)
}
func isTrace() bool { return false }
func logTrace(args ...interface{}) {
	if isTrace() {
		fmt.Fprintln(LogWriter, args)
	}
}
