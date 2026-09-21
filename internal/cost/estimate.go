package cost

import (
	"math"
	"slices"
	"strings"

	"github.com/taylor-swanson/package-tool/pkg/fleetpkg"
)

var wsReplacer = strings.NewReplacer(" ", "", "\n", "", "\t", "", "\r", "")

func EstimateProcessor(processor *fleetpkg.Processor) ProcessorReport {
	report := ProcessorReport{
		Type: processor.Type,
		Tag:  processor.GetAttributeStringOr("tag", ""),
		Cost: 1,
	}

	// TODO: Per-processor specific cost estimations...
	switch processor.Type {
	case "dissect":
		report.Cost = 2
	case "json":
		report.Cost = 5
	case "kv":
		report.Cost = 5
	case "grok":
		report.Cost = 7
		if raw, ok := processor.GetAttribute("patterns"); ok {
			if rawPatterns, ok := raw.([]any); ok {
				for _, rawPattern := range rawPatterns {
					if pattern, ok := rawPattern.(string); ok {
						report.Cost += stringCostByBytes(pattern)
					}
				}
			}
		}
	case "geoip":
		report.Cost = 10
	case "script":
		report.Cost = 10
		if source := processor.GetAttributeStringOr("source", ""); source != "" {
			report.Cost += stringCostByBytes(source)
		}
	}

	if conditional := processor.GetAttributeStringOr("if", ""); conditional != "" {
		report.Cost += stringCostByBytes(conditional)
	}

	return report
}

func EstimatePipeline(pipeline *fleetpkg.Pipeline) PipelineReport {
	var report PipelineReport

	for _, v := range pipeline.Processors {
		processorReport := EstimateProcessor(v)
		report.Cost += processorReport.Cost
		report.Processors = append(report.Processors, processorReport)
	}

	return report
}

func EstimateDataStream(dataStream *fleetpkg.DataStream) DataStreamReport {
	report := DataStreamReport{
		Pipelines: map[string]PipelineReport{},
	}

	for k, v := range dataStream.Pipelines {
		pipelineReport := EstimatePipeline(v)
		report.Cost += pipelineReport.Cost
		report.Pipelines[k] = pipelineReport
	}

	return report
}

func EstimatePackage(pkg *fleetpkg.Package, dataStreamFilters ...string) PackageReport {
	report := PackageReport{
		DataStreams: map[string]DataStreamReport{},
	}

	for k, v := range pkg.DataStreams {
		if len(dataStreamFilters) > 0 && !slices.Contains(dataStreamFilters, k) {
			continue
		}

		dataStreamReport := EstimateDataStream(v)
		report.Cost += dataStreamReport.Cost
		report.DataStreams[k] = dataStreamReport
	}

	return report
}

func stringCostByBytes(s string) int {
	return int(math.Ceil(float64(len(wsReplacer.Replace(s))) * 0.1))
}
