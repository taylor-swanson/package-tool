package cost

import "time"

type ProcessorReport struct {
	Type string `json:"type"`
	Tag  string `json:"tag"`
	Cost int    `json:"cost"`
}

type PipelineReport struct {
	Cost       int               `json:"cost"`
	Processors []ProcessorReport `json:"-"`
}

type DataStreamReport struct {
	Cost      int                       `json:"cost"`
	Pipelines map[string]PipelineReport `json:"pipelines,omitempty"`
}

type PackageReport struct {
	Cost        int                         `json:"cost"`
	DataStreams map[string]DataStreamReport `json:"data_streams,omitempty"`
}

type Report struct {
	Timestamp time.Time                `json:"timestamp"`
	Packages  []string                 `json:"packages,omitempty"`
	Reports   map[string]PackageReport `json:"reports,omitempty"`
}
