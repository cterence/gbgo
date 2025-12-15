package log

import (
	"fmt"
)

var DebugEnabled bool

func Debug(format string, args ...any) {
	if DebugEnabled {
		fmt.Printf(format+"\n", args...)
	}
}
