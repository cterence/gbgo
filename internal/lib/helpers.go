package lib

import "fmt"

func Assert(condition bool, msg string, args ...any) {
	if !condition {
		panic(fmt.Sprintf(msg, args...))
	}
}
