package log

import (
	"os"
)

// Pipe reads all incoming traffic from a named pipe and sends it directly to the output

func (h *Harvester) openPipe() error {

	fifo, err := os.OpenFile(h.state.Source, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return err
	}

	// Here we reuse the Pipe struct already defined by Stdin
	h.source = Pipe{File: fifo}

	h.encoding, err = h.encodingFactory(h.source)

	return err
}
