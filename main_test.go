package main

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()
	}

	os.Exit(m.Run())
}

func Test_Blargg_CPU_Serial(t *testing.T) {
	duration := 5 * time.Second
	timeout := time.After(duration)
	done := make(chan bool)
	expected := []byte("cpu_instrs\n\n01:ok  02:ok  03:ok  04:ok  05:ok  06:ok  07:ok  08:ok  09:ok  10:ok  11:ok  \n\nPassed all tests")
	cmd := exec.Command(os.Args[0], "./sub/gb-test-roms/cpu_instrs/cpu_instrs.gb", "--hl", "--ps", "--ns")

	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	r, w := io.Pipe()
	cmd.Stdout = w

	assert.NoError(t, cmd.Start())

	go func() {
		buf := make([]uint8, 1)
		for i, want := range expected {
			n, err := r.Read(buf)
			assert.NoError(t, err)
			assert.Equal(t, 1, n)
			assert.Equal(t, string(want), string(buf[0]), "wrong byte received at %d", i)
		}

		done <- true
	}()

	select {
	case <-timeout:
		t.Fatalf("test timed out after %s", duration)
	case <-done:
		assert.NoError(t, cmd.Process.Kill())
	}
}
