package treesql

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	registry *prometheus.Registry

	// Counters
	nextConnectionID prometheus.CounterFunc

	// Gauges
	openConnections        prometheus.GaugeFunc
	openChannels           prometheus.GaugeFunc
	filteredTableListeners prometheus.GaugeFunc
	wholeTableListeners    prometheus.GaugeFunc
	recordListeners        prometheus.GaugeFunc

	// Latency histograms
	selectLatency        prometheus.Histogram
	insertLatency        prometheus.Histogram
	updateLatency        prometheus.Histogram
	liveQueryPushLatency prometheus.Histogram
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
		openConnections: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "open_connections",
				Help: "number of connections currently open",
			},
			func() float64 {
				return float64(len(db.Connections))
			},
		),
		openChannels: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
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
		recordListeners: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
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
		filteredTableListeners: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
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
		wholeTableListeners: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
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
		selectLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "select_latency_ns",
				Help: "latency to return initial results of SELECT statements",
			},
		),
		insertLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "insert_latency_ns",
				Help: "latency to execute an INSERT statement",
			},
		),
		updateLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "update_latency_ns",
				Help: "latency to execute an UPDATE statement",
			},
		),
		liveQueryPushLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name: "live_query_push_latency_ns",
				Help: "latency to push updates to live queries on an insert, update, or delete",
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
	m.registry.MustRegister(m.selectLatency)
	m.registry.MustRegister(m.insertLatency)
	m.registry.MustRegister(m.updateLatency)
	m.registry.MustRegister(m.liveQueryPushLatency)
	return m
}
