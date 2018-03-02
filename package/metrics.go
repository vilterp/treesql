package treesql

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	registry *prometheus.Registry

	nextConnectionID prometheus.CounterFunc
	openConnections  prometheus.CounterFunc
	openChannels     prometheus.CounterFunc
	tableListeners   prometheus.CounterFunc
	recordListeners  prometheus.CounterFunc
}

func NewMetrics(db *Database) *Metrics {
	m := &Metrics{
		nextConnectionID: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "next_connection_id",
				Help: "number of connections to this server over its lifetime",
			},
			func() float64 {
				return float64(db.NextConnectionID)
			},
		),
		openConnections: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "open_connections",
				Help: "number of connections currently open",
			},
			func() float64 {
				return float64(len(db.Connections))
			},
		),
		openChannels: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "open_channels",
				Help: "number of channels currently open across all connections",
			},
			func() float64 {
				// TODO: synchronize access to db.Connections...
				// TODO: make this not O(connections) somehow...
				// but I also don't want two sources of truth
				count := 0
				for _, conn := range db.Connections {
					count += len(conn.Channels)
				}
				return float64(count)
			},
		),
	}
	m.registry = prometheus.NewPedanticRegistry()
	m.registry.MustRegister(m.nextConnectionID)
	m.registry.MustRegister(m.openConnections)
	m.registry.MustRegister(m.openChannels)
	return m
}
