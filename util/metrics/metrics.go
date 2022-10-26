package metrics

import (
	"ethattacksim/interfaces"
	"ethattacksim/util/file"
	"github.com/rcrowley/go-metrics"
	"io"
	"time"
)

var config *file.Config
var registry metrics.Registry

func Initialize(conf *file.Config) {
	config = conf
	registry = metrics.NewRegistry()
}

func NameFormat(name interfaces.IMetricName, id string) string {
	return name.String() + "_" + id
}

func Timer(name string, value time.Duration) {
	if config == nil {
		metrics.GetOrRegisterTimer(name+"_Timer", registry).Update(value)
	} else {
		if config.UseMetrics() {
			metrics.GetOrRegisterTimer(name+"_Timer", registry).Update(value)
		}
	}
}

func Gauge(name string, value int64) {
	if config == nil {
		metrics.GetOrRegisterGauge(name+"_Gauge", registry).Update(value)
	} else {
		if config.UseMetrics() {
			metrics.GetOrRegisterGauge(name+"_Gauge", registry).Update(value)
		}
	}
}

func FloatGauge(name string, value float64) {
	if config == nil {
		metrics.GetOrRegisterGaugeFloat64(name+"_FloatGauge", registry).Update(value)
	} else {
		if config.UseMetrics() {
			metrics.GetOrRegisterGaugeFloat64(name+"_FloatGauge", registry).Update(value)
		}
	}
}

func Counter(name string, value int64) {
	if config == nil {
		if value > 0 {
			metrics.GetOrRegisterCounter(name+"_Counter", registry).Inc(value)
		} else {
			metrics.GetOrRegisterCounter(name+"_Counter", registry).Dec(value * -1)
		}
	} else {
		if config.UseMetrics() {
			if value > 0 {
				metrics.GetOrRegisterCounter(name+"_Counter", registry).Inc(value)
			} else {
				metrics.GetOrRegisterCounter(name+"_Counter", registry).Dec(value * -1)
			}
		}
	}
}

func WriteToFile(writer io.Writer) {
	metrics.WriteJSONOnce(registry, writer)
}
