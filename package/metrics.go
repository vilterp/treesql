package treesql

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	registry *prometheus.Registry

	nextConnectionID       prometheus.CounterFunc
	openConnections        prometheus.CounterFunc
	openChannels           prometheus.CounterFunc
	filteredTableListeners prometheus.CounterFunc
	wholeTableListeners    prometheus.CounterFunc
	recordListeners        prometheus.CounterFunc
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
		recordListeners: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "record_listeners",
				Help: "number of record listeners across the database",
			},
			func() float64 {
				// TODO: synchronize access to listeners
				count := 0
				for _, table := range db.Schema.Tables {
					for _, listenerList := range table.LiveQueryInfo.RecordListeners {
						count += listenerList.numListeners
					}
				}
				return float64(count)
			},
		),
		filteredTableListeners: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "filtered_table_listeners",
				Help: "number of filtered table listeners across the database",
			},
			func() float64 {
				// TODO: synchronize access to listeners
				count := 0
				for _, table := range db.Schema.Tables {
					for _, listenersForCol := range table.LiveQueryInfo.TableListeners {
						for _, listeners := range listenersForCol {
							count += listeners.NumListeners()
						}
					}
				}
				return float64(count)
			},
		),
		wholeTableListeners: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "whole_table_listeners",
				Help: "number of whole table listeners across the database",
			},
			func() float64 {
				// TODO: synchronize access to listeners
				count := 0
				for _, table := range db.Schema.Tables {
					count += table.LiveQueryInfo.WholeTableListeners.NumListeners()
				}
				return float64(count)
			},
		),
	}
	m.registry = prometheus.NewPedanticRegistry()
	m.registry.MustRegister(m.nextConnectionID)
	m.registry.MustRegister(m.openConnections)
	m.registry.MustRegister(m.openChannels)
	m.registry.MustRegister(m.recordListeners)
	m.registry.MustRegister(m.filteredTableListeners)
	m.registry.MustRegister(m.wholeTableListeners)
	return m
}
