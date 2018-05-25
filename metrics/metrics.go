// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package metrics

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/medibloc/go-medibloc/medlet/pb"
	"github.com/medibloc/go-medibloc/util/logging"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
)

const (
	interval = 2 * time.Second
	chainID  = "chainID"
	// MetricsEnabledFlag metrics enable flag
	MetricsEnabledFlag = "metrics"
)

var (
	enable = false
	quitCh chan (bool)
)

// Medlet interface breaks cycle import dependency.
type Medlet interface {
	Config() *medletpb.Config
}

func init() {
	for _, arg := range os.Args {
		if strings.TrimLeft(arg, "-") == MetricsEnabledFlag {
			EnableMetrics()
			return
		}
	}
}

// EnableMetrics enable the metrics service
func EnableMetrics() {
	enable = true
	exp.Exp(metrics.DefaultRegistry)
}

// Start metrics monitor
func Start(med Medlet) {
	logging.Info("Starting Metrics...")

	go (func() {
		tags := make(map[string]string)
		metricsConfig := med.Config().Stats.MetricsTags
		for _, v := range metricsConfig {
			values := strings.Split(v, ":")
			if len(values) != 2 {
				continue
			}
			tags[values[0]] = values[1]
		}
		tags[chainID] = fmt.Sprintf("%d", med.Config().Global.ChainId)
		go collectSystemMetrics()
		InfluxDBWithTags(metrics.DefaultRegistry, interval, med.Config().Stats.Influxdb.Host, med.Config().Stats.Influxdb.Db, med.Config().Stats.Influxdb.User, med.Config().Stats.Influxdb.Password, tags)

		logging.Info("Started Metrics.")

	})()

	logging.Info("Started Metrics.")
}

func collectSystemMetrics() {
	memstats := make([]*runtime.MemStats, 2)
	for i := 0; i < len(memstats); i++ {
		memstats[i] = new(runtime.MemStats)
	}

	allocs := metrics.GetOrRegisterMeter("system_allocs", nil)
	// totalAllocs := metrics.GetOrRegisterMeter("system_total_allocs", nil)
	sys := metrics.GetOrRegisterMeter("system_sys", nil)
	frees := metrics.GetOrRegisterMeter("system_frees", nil)
	heapInuse := metrics.GetOrRegisterMeter("system_heapInuse", nil)
	stackInuse := metrics.GetOrRegisterMeter("system_stackInuse", nil)
	for i := 1; ; i++ {
		select {
		case <-quitCh:
			return
		default:
			runtime.ReadMemStats(memstats[i%2])
			allocs.Mark(int64(memstats[i%2].Alloc - memstats[(i-1)%2].Alloc))
			sys.Mark(int64(memstats[i%2].Sys - memstats[(i-1)%2].Sys))
			frees.Mark(int64(memstats[i%2].Frees - memstats[(i-1)%2].Frees))
			heapInuse.Mark(int64(memstats[i%2].HeapInuse - memstats[(i-1)%2].HeapInuse))
			stackInuse.Mark(int64(memstats[i%2].StackInuse - memstats[(i-1)%2].StackInuse))
			time.Sleep(2 * time.Second)
		}
	}

}

// Stop metrics monitor
func Stop() {
	logging.Info("Stopping Metrics...")

	quitCh <- true
}

// NewCounter create a new metrics Counter
func NewCounter(name string) metrics.Counter {
	if !enable {
		return new(metrics.NilCounter)
	}
	return metrics.GetOrRegisterCounter(name, metrics.DefaultRegistry)
}

// NewMeter create a new metrics Meter
func NewMeter(name string) metrics.Meter {
	if !enable {
		return new(metrics.NilMeter)
	}
	return metrics.GetOrRegisterMeter(name, metrics.DefaultRegistry)
}

// NewTimer create a new metrics Timer
func NewTimer(name string) metrics.Timer {
	if !enable {
		return new(metrics.NilTimer)
	}
	return metrics.GetOrRegisterTimer(name, metrics.DefaultRegistry)
}

// NewGauge create a new metrics Gauge
func NewGauge(name string) metrics.Gauge {
	if !enable {
		return new(metrics.NilGauge)
	}
	return metrics.GetOrRegisterGauge(name, metrics.DefaultRegistry)
}

// NewHistogramWithUniformSample create a new metrics History with Uniform Sample algorithm.
func NewHistogramWithUniformSample(name string, reservoirSize int) metrics.Histogram {
	if !enable {
		return new(metrics.NilHistogram)
	}
	return metrics.GetOrRegisterHistogram(name, nil, metrics.NewUniformSample(reservoirSize))
}
