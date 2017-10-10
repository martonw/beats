package pipe

import (
	"fmt"
	"os"
	"syscall"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/prospector/log"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := prospector.Register("pipe", NewProspector)
	if err != nil {
		panic(err)
	}
}

// Prospector is a prospector for pipe
type Prospector struct {
	harvester *log.Harvester
	started   bool
	config    config
	cfg       *common.Config
	outlet    channel.Outleter
	registry  *harvester.Registry
}

// fifo handler used to set pipe size
var fifo *os.File;

// NewProspector creates a new pipe prospector
// This prospector contains one harvester which is reading from the given named pipe
func NewProspector(cfg *common.Config, outlet channel.Factory, context prospector.Context) (prospector.Prospectorer, error) {
	config := defaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	out, err := outlet(cfg)
	if err != nil {
		return nil, err
	}

	p := &Prospector{
		started:  false,
		config:   config,
		cfg:      cfg,
		outlet:   out,
		registry: harvester.NewRegistry(),
	}

	p.harvester, err = p.createHarvester(file.State{Source: config.Path})
	if err != nil {
		return nil, fmt.Errorf("Error initializing pipe harvester: %v", err)
	}

	return p, nil
}

// Run runs the prospector
func (p *Prospector) Run() {
	// Make sure pipe harvester is only started once
	if !p.started {
		err := p.harvester.Setup()
		if err != nil {
			logp.Err("Error setting up pipe harvester: %s", err)
			return
		}
		p.registry.Start(p.harvester)
		p.started = true
	}
}


// createHarvester creates a new harvester instance from the given state
func (p *Prospector) createHarvester(state file.State) (*log.Harvester, error) {

	// Each harvester gets its own copy of the outlet
	h, err := log.NewHarvester(
		p.cfg,
		state, nil, nil,
		p.outlet,
	)

	// check pipsize config
	if p.config.PipeSize > 65536 {
		logp.Info("Custom pipe size set for pipe input. Requested size: %d", p.config.PipeSize)

		// The fifo handler remains open until the Harvester is stopped. This is
		// necessary, otherwise the PIPE SIZE change would not be effective after all
		// handlers are closed and reopened later (when the Harvester starts the reading)
		var err error
		fifo, err = os.OpenFile(state.Source, os.O_RDWR, os.ModeNamedPipe)
		if err != nil {
			return nil, err
		}

		reportedPipeSize, _, errCode := syscall.Syscall(syscall.SYS_FCNTL, fifo.Fd(), syscall.F_SETPIPE_SZ, uintptr(p.config.PipeSize))
		// reportedPipeSize, _, errCode := syscall.Syscall(syscall.SYS_FCNTL, fifo.Fd(), syscall.F_GETPIPE_SZ, 0)
		if errCode != 0 {
			logp.Err("Could not set pipe size to the requested number. Syscall error code=%d", errCode)
		}
		// Linux kernel may round up the buffer size, so larger than requested is ok.
		if p.config.PipeSize > int(reportedPipeSize) {
			logp.Err("Could not set pipe size to the requested number. Requested=%d, Reported=%d", p.config.PipeSize, int(reportedPipeSize))
		} else {
			logp.Info("Pipe size successfully set to %d.", reportedPipeSize)
		}
	}

	return h, err
}

// Wait waits for completion of the prospector.
func (p *Prospector) Wait() {}

// Stop stops the prospector.
func (p *Prospector) Stop() {
	p.outlet.Close()
	fifo.Close()
}
