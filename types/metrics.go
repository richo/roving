package types

import (
	"fmt"

	"github.com/getsentry/raven-go"
	"github.com/stripe/veneur/ssf"
	"github.com/stripe/veneur/trace"
	"github.com/stripe/veneur/trace/metrics"
)

func SubmitMetricCount(name string, value float32, tags map[string]string) {
	submitMetric(ssf.Count(name, value, tags))
}

func SubmitMetricGauge(name string, value float32, tags map[string]string) {
	submitMetric(ssf.Gauge(name, value, tags))
}

func submitMetric(metric *ssf.SSFSample) {
	err := metrics.ReportOne(trace.DefaultClient, metric)
	if err != nil {
		ts := metric.Tags
		ts["metric_name"] = metric.Name
		ts["metric_value"] = fmt.Sprintf("%f", metric.Value)
		raven.CaptureError(err, ts)
	}
}
