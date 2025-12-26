package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()

		return
	}

	os.Exit(m.Run())
}

func runCPUTestGBD(t *testing.T, testFileName string) {
	t.Skip("no ppu interrupt support")
	t.Parallel()

	testNum := strings.Split(testFileName, "-")[0]
	testNumInt, err := strconv.Atoi(testNum)
	assert.NoError(t, err)

	cmd1 := exec.Command(os.Args[0], "./sub/gb-test-roms/cpu_instrs/individual/"+testFileName, "--gbd", "--hl")
	cmd2 := exec.Command("./sub/gameboy-doctor/gameboy-doctor", "-", "cpu_instrs", strconv.Itoa(testNumInt))

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

func Test_Blargg_CPU_GBD(t *testing.T) {
	t.Run("01-special.gb", func(t *testing.T) {
		runCPUTestGBD(t, "01-special.gb")
	})

	t.Run("02-interrupts.gb", func(t *testing.T) {
		runCPUTestGBD(t, "02-interrupts.gb")
	})

	t.Run("03-op sp,hl.gb", func(t *testing.T) {
		runCPUTestGBD(t, "03-op sp,hl.gb")
	})

	t.Run("04-op r,imm.gb", func(t *testing.T) {
		runCPUTestGBD(t, "04-op r,imm.gb")
	})

	t.Run("05-op rp.gb", func(t *testing.T) {
		runCPUTestGBD(t, "05-op rp.gb")
	})

	t.Run("06-ld r,r.gb", func(t *testing.T) {
		runCPUTestGBD(t, "06-ld r,r.gb")
	})

	t.Run("07-jr,jp,call,ret,rst.gb", func(t *testing.T) {
		runCPUTestGBD(t, "07-jr,jp,call,ret,rst.gb")
	})

	t.Run("08-misc instrs.gb", func(t *testing.T) {
		runCPUTestGBD(t, "08-misc instrs.gb")
	})

	t.Run("09-op r,r.gb", func(t *testing.T) {
		runCPUTestGBD(t, "09-op r,r.gb")
	})

	t.Run("10-bit ops.gb", func(t *testing.T) {
		runCPUTestGBD(t, "10-bit ops.gb")
	})

	t.Run("11-op a,(hl).gb", func(t *testing.T) {
		runCPUTestGBD(t, "11-op a,(hl).gb")
	})
}

func Test_Blargg_CPU_Serial(t *testing.T) {
	expected := []byte("cpu_instrs\n\n01:ok  02:ok  03:ok  04:ok  05:ok  06:ok  07:ok  08:ok  09:ok  10:ok  11:ok  \n\nPassed all tests")

	cmd := exec.Command(os.Args[0], "./sub/gb-test-roms/cpu_instrs/cpu_instrs.gb", "--hl", "--ps", "--ns")

	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	r, w := io.Pipe()
	cmd.Stdout = w
	assert.NoError(t, cmd.Start())

	buf := make([]uint8, 1)

	for i, want := range expected {
		n, err := r.Read(buf)
		assert.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, string(want), string(buf[0]), "wrong byte received at %d", i)
	}

	assert.NoError(t, cmd.Process.Kill())
}
