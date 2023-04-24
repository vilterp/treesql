package treesql

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
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
	selectLatency        prometheus.Summary
	insertLatency        prometheus.Summary
	updateLatency        prometheus.Summary
	liveQueryPushLatency prometheus.Summary

	scanLatency   prometheus.Summary
	lookupLatency prometheus.Summary
}

func newMetrics(db *Database) *metrics {
	m := &metrics{
		nextConnectionID: prometheus.NewCounterFunc(
			prometheus.CounterOpts{
				Name: "next_connection_id",
				Help: "number of connections to this server over its lifetime",
			},
			func() float64 {
				return float64(db.nextConnectionID)
			},
		),
		openConnections: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "open_connections",
				Help: "number of connections currently open",
			},
			func() float64 {
				return float64(len(db.connections))
			},
		),
		openChannels: prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "open_channels",
				Help: "number of channels currently open across all connections",
			},
			func() float64 {
				// TODO: synchronize access to db.connections...
				// TODO: make this not O(connections) somehow...
				// but I also don't want two sources of truth
				count := 0
				for _, conn := range db.connections {
					count += len(conn.channels)
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
				for _, table := range db.schema.tables {
					table.liveQueryInfo.mu.RLock()
					defer table.liveQueryInfo.mu.RUnlock()

					for _, listenerList := range table.liveQueryInfo.mu.RecordListeners {
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
				for _, table := range db.schema.tables {
					table.liveQueryInfo.mu.RLock()
					defer table.liveQueryInfo.mu.RUnlock()

					for _, listenersForCol := range table.liveQueryInfo.mu.TableListeners {
						for _, listeners := range listenersForCol {
							count += listeners.getNumListeners()
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
				for _, table := range db.schema.tables {
					table.liveQueryInfo.mu.RLock()
					defer table.liveQueryInfo.mu.RUnlock()

					count += table.liveQueryInfo.mu.WholeTableListeners.getNumListeners()
				}
				return float64(count)
			},
		),
		selectLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "select_latency_ns",
				Help: "latency to return initial results of SELECT statements",
			},
		),
		insertLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "insert_latency_ns",
				Help: "latency to execute an INSERT statement",
			},
		),
		updateLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "update_latency_ns",
				Help: "latency to execute an UPDATE statement",
			},
		),
		liveQueryPushLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "live_query_push_latency_ns",
				Help: "latency to push updates to live queries on an insert, update, or delete",
			},
		),
		scanLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "scan_latency_ns",
				Help: "latency to scan a table",
			},
		),
		lookupLatency: prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: "lookup_latency_ns",
				Help: "latency to look up a single record in a table",
			},
		),
	}
	m.registry = prometheus.NewPedanticRegistry()
	reg := m.registry

	reg.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	reg.MustRegister(prometheus.NewGoCollector())

	reg.MustRegister(m.nextConnectionID)
	reg.MustRegister(m.openConnections)
	reg.MustRegister(m.openChannels)
	reg.MustRegister(m.recordListeners)
	reg.MustRegister(m.filteredTableListeners)
	reg.MustRegister(m.wholeTableListeners)
	reg.MustRegister(m.selectLatency)
	reg.MustRegister(m.insertLatency)
	reg.MustRegister(m.updateLatency)
	reg.MustRegister(m.liveQueryPushLatency)
	reg.MustRegister(m.scanLatency)
	reg.MustRegister(m.lookupLatency)
	return m
}
