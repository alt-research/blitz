package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type FpMetrics struct {
	orbitFinalizedHeight        prometheus.Gauge
	orbitBabylonFinalizedHeight prometheus.Gauge
	fpCommittedHeight           prometheus.Gauge
}

// Declare a package-level variable for sync.Once to ensure metrics are registered only once
var fpMetricsRegisterOnce sync.Once

// Declare a variable to hold the instance of FpMetrics
var fpMetricsInstance *FpMetrics

// NewFpMetrics initializes and registers the metrics, using sync.Once to ensure it's done only once
func NewFpMetrics() *FpMetrics {
	fpMetricsRegisterOnce.Do(func() {
		fpMetricsInstance = &FpMetrics{
			orbitFinalizedHeight: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "orbit_finalized_height",
				Help: "orbit finalized height by nodes",
			}),
			orbitBabylonFinalizedHeight: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "orbit_babylon_finalized_height",
				Help: "orbit finalized height by finality provider",
			}),
			fpCommittedHeight: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "fp_committed_height",
				Help: "The orbit height committed by fp",
			}),
		}

		// Register the metrics with Prometheus
		prometheus.MustRegister(fpMetricsInstance.orbitFinalizedHeight)
		prometheus.MustRegister(fpMetricsInstance.orbitBabylonFinalizedHeight)
		prometheus.MustRegister(fpMetricsInstance.fpCommittedHeight)
	})
	return fpMetricsInstance
}

func (fm *FpMetrics) RecordCommittedHeight(fpBtcPkHex string, height uint64) {
	fm.fpCommittedHeight.Set(float64(height))
}

func (fm *FpMetrics) RecordOrbitFinalizedHeight(fpBtcPkHex string, height uint64) {
	fm.orbitFinalizedHeight.Set(float64(height))
}

func (fm *FpMetrics) RecordOrbitBabylonFinalizedHeight(fpBtcPkHex string, height uint64, Hash []byte) {
	fm.orbitBabylonFinalizedHeight.Set(float64(height))
}
