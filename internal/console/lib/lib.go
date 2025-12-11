package lib

import "fmt"

func Assert(cond bool, err error) {
	if !cond {
		panic(fmt.Errorf("assertion failed: %w", err))
	}
}

func Must[T any](ret T, err error) T {
	if err != nil {
		panic(err)
	}

	return ret
}
