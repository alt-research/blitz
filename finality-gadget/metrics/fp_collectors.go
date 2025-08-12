package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type FpMetrics struct {
	fpBabylonAddressBalances *prometheus.GaugeVec
}

// Declare a package-level variable for sync.Once to ensure metrics are registered only once
var fpMetricsRegisterOnce sync.Once

// Declare a variable to hold the instance of FpMetrics
var fpMetricsInstance *FpMetrics

// NewFpMetrics initializes and registers the metrics, using sync.Once to ensure it's done only once
func NewFpMetrics() *FpMetrics {
	fpMetricsRegisterOnce.Do(func() {
		fpMetricsInstance = &FpMetrics{
			fpBabylonAddressBalances: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "fp_babylon_address_balances",
				Help: "Current Balance of a finality provider 's babylon address",
			}, []string{"fp_address"}),
		}

		// Register the metrics with Prometheus
		prometheus.MustRegister(fpMetricsInstance.fpBabylonAddressBalances)
	})
	return fpMetricsInstance
}

func (fm *FpMetrics) RecordFpBalance(address string, balance float64) {
	fm.fpBabylonAddressBalances.WithLabelValues(address).Set(balance)
}
