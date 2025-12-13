package lib

func Must[T any](ret T, err error) T {
	if err != nil {
		panic(err)
	}

	return ret
}
