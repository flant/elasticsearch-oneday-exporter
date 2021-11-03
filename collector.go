package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	labels    = []string{"index", "cluster", "project"}
	indexSize = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "indices_store", "size_bytes_primary"), "Size for each index in today", labels, nil,
	)
	docsCount = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "indices_docs", "total"), "Count docs for each index in today", labels, nil,
	)
)

type Collector struct{}

func (i *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- indexSize
	ch <- docsCount
}

func (i *Collector) Collect(ch chan<- prometheus.Metric) {
	client, err := NewClient([]string{fmt.Sprintf("%s:%s", *esUrl, *esPort)}, true)
	if err != nil {
		log.Fatal(err)
	}

	indices, err := client.GetIndices([]string{fmt.Sprintf("*-%s", time.Now().Format("2006.01.02"))})
	if err != nil {
		log.Fatal(err)
	}

	for name, val := range indices {
		index := val.(map[string]interface{})

		total := index["total"].(map[string]interface{})
		docs := total["docs"].(map[string]interface{})
		count := docs["count"].(float64)
		ch <- prometheus.MustNewConstMetric(docsCount, prometheus.GaugeValue, count, name, *clusterName, *projectName)

		primaries := index["primaries"].(map[string]interface{})
		store := primaries["store"].(map[string]interface{})
		size := store["size_in_bytes"].(float64)
		ch <- prometheus.MustNewConstMetric(indexSize, prometheus.GaugeValue, size, name, *clusterName, *projectName)

	}
}
