package net

import (
	"fmt"

	metrics "github.com/medibloc/go-medibloc/metrics"
)

// Metrics map for different in/out network msg types
var (
	metricsPacketsIn = metrics.NewMeter("med.net.packets.in")
	metricsBytesIn   = metrics.NewMeter("med.net.bytes.in")

	metricsPacketsOut = metrics.NewMeter("med.net.packets.out")
	metricsBytesOut   = metrics.NewMeter("med.net.bytes.out")
)

func metricsPacketsInByMessageName(messageName string, size uint64) {
	meter := metrics.NewMeter(fmt.Sprintf("med.net.packets.in.%s", messageName))
	meter.Mark(1)

	meter = metrics.NewMeter(fmt.Sprintf("med.net.bytes.in.%s", messageName))
	meter.Mark(int64(size))
}

func metricsPacketsOutByMessageName(messageName string, size uint64) {
	meter := metrics.NewMeter(fmt.Sprintf("med.net.packets.out.%s", messageName))
	meter.Mark(1)

	meter = metrics.NewMeter(fmt.Sprintf("med.net.bytes.out.%s", messageName))
	meter.Mark(int64(size))
}
