package log

import (
	"fmt"
	"os"
)

var DebugEnabled bool

func Debug(format string, args ...any) {
	if DebugEnabled {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
