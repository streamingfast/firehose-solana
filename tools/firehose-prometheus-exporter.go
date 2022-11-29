package tools

import (
	sftools "github.com/streamingfast/sf-tools"
)

func init() {
	prometheusExporterCmd := sftools.GetFirehosePrometheusExporterCmd(zlog, tracer, transformsSetter)
	Cmd.AddCommand(prometheusExporterCmd)
}
