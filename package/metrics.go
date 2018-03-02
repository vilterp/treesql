package treesql

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	registry *prometheus.Registry

	nextConnectionID prometheus.CounterFunc
}

func NewMetrics(db *Database) *Metrics {
	m := &Metrics{
		// TODO: open connections...
		nextConnectionID: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "next_connection_id",
				Help: "number of connections to this server over its lifetime",
			},
			func() float64 {
				return float64(db.NextConnectionID)
			},
		),
	}
	m.registry = prometheus.NewPedanticRegistry()
	m.registry.MustRegister(m.nextConnectionID)
	return m
}
