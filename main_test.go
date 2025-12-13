package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/cterence/gbgo/internal/console/lib"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()

		return
	}

	os.Exit(m.Run())
}

func runCPUTest(t *testing.T, testFileName string) {
	t.Parallel()

	testNum := strings.Split(testFileName, "-")[0]
	cmd1 := exec.Command(os.Args[0], "./sub/gb-test-roms/cpu_instrs/individual/"+testFileName, "--gbd")
	cmd2 := exec.Command("./sub/gameboy-doctor/gameboy-doctor", "-", "cpu_instrs", strconv.Itoa(lib.Must(strconv.Atoi(testNum))))

	var err error

	cmd2.Stdin, err = cmd1.StdoutPipe()
	assert.NoError(t, err)

	var b2 bytes.Buffer

	cmd2.Stdout = &b2

	cmd1.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	assert.NoError(t, cmd2.Start())
	assert.NoError(t, cmd1.Start())
	assert.NoError(t, cmd2.Wait())
	assert.NoError(t, cmd1.Process.Kill())

	out, err := io.ReadAll(&b2)
	assert.NoError(t, err)
	t.Logf("Output from %s:\n%s", testFileName, string(out))
	assert.Contains(t, string(out), "SUCCESS")
}

func Test_CPU(t *testing.T) {
	t.Run("01-special.gb", func(t *testing.T) {
		runCPUTest(t, "01-special.gb")
	})

	t.Run("02-interrupts.gb", func(t *testing.T) {
		runCPUTest(t, "02-interrupts.gb")
	})

	t.Run("03-op sp,hl.gb", func(t *testing.T) {
		runCPUTest(t, "03-op sp,hl.gb")
	})

	t.Run("04-op r,imm.gb", func(t *testing.T) {
		runCPUTest(t, "04-op r,imm.gb")
	})

	t.Run("05-op rp.gb", func(t *testing.T) {
		runCPUTest(t, "05-op rp.gb")
	})

	t.Run("06-ld r,r.gb", func(t *testing.T) {
		runCPUTest(t, "06-ld r,r.gb")
	})

	t.Run("07-jr,jp,call,ret,rst.gb", func(t *testing.T) {
		runCPUTest(t, "07-jr,jp,call,ret,rst.gb")
	})

	t.Run("08-misc instrs.gb", func(t *testing.T) {
		runCPUTest(t, "08-misc instrs.gb")
	})

	t.Run("09-op r,r.gb", func(t *testing.T) {
		runCPUTest(t, "09-op r,r.gb")
	})

	t.Run("10-bit ops.gb", func(t *testing.T) {
		runCPUTest(t, "10-bit ops.gb")
	})

	t.Run("11-op a,(hl).gb", func(t *testing.T) {
		runCPUTest(t, "11-op a,(hl).gb")
	})
}
