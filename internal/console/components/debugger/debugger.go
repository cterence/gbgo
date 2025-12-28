package debugger

import (
	"fmt"
	"io"
)

const (
	DEBUGGER_SIZE = 2048
)

type Debugger struct {
	writer           io.Writer
	traces           [DEBUGGER_SIZE]string
	head, tail, size int
}

func (d *Debugger) Init(w io.Writer) {
	d.traces = [DEBUGGER_SIZE]string{}
	d.head = 0
	d.tail = 0
	d.size = 0
	d.writer = w

	go d.WriteTraces()
}

func (d *Debugger) Push(trace string) {
	d.traces[d.tail] = trace
	d.tail = (d.tail + 1) % DEBUGGER_SIZE
	d.size++

	if d.size > DEBUGGER_SIZE {
		d.size = DEBUGGER_SIZE
	}
}

func (d *Debugger) Pop() (string, bool) {
	if d.size == 0 {
		return "", false
	}

	trace := d.traces[d.head]
	d.head = (d.head + 1) % DEBUGGER_SIZE
	d.size--

	return trace, true
}

func (d *Debugger) WriteTraces() {
	for {
		trace, ok := d.Pop()

		if !ok {
			continue
		}

		_, err := d.writer.Write([]byte(trace + "\n"))
		if err != nil {
			fmt.Printf("failed to write to debugger writer: %v\n", err)
		}
	}
}
