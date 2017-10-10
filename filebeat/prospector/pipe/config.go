package pipe

import (
	"github.com/elastic/beats/filebeat/harvester"
)

var defaultConfig = config{

	ForwarderConfig: harvester.ForwarderConfig{
		Type: "pipe",
	},
  PipeSize: 65536,
}

type config struct {
	harvester.ForwarderConfig `config:",inline"`
	Path                      string        `config:"path" validate:"required"`
  PipeSize                  int           `config:"pipesize" validate: "min=65536"`
}
