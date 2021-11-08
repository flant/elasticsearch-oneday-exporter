package main

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	client *Client

	indexSize *prometheus.Desc
	docsCount *prometheus.Desc
}

func NewCollector(address, project string, insecure bool) (*Collector, error) {
	namespace := "oneday_elasticsearch"
	labels := []string{"index"}

	client, err := NewClient([]string{address}, insecure)
	if err != nil {
		return nil, fmt.Errorf("error creating the client: %v", err)
	}

	info, err := client.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("error getting cluster info: %v", err)
	}
	log.Infof("Cluster info: %v", info)

	cluster := info["cluster_name"].(string)

	constLabels := prometheus.Labels{
		"cluster": cluster,
		"project": project,
	}

	return &Collector{client: client,
		indexSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_store", "size_bytes_primary"),
			"Size of each index to date", labels, constLabels,
		),
		docsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_docs", "total"),
			"Count of docs for each index to date", labels, constLabels,
		),
	}, nil
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.indexSize
	ch <- c.docsCount
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	indices, err := c.client.GetIndices([]string{fmt.Sprintf("*-%s", time.Now().Format("2006.01.02"))})
	if err != nil {
		log.Fatal("error getting indices stats: ", err)
	}

	for index, v := range indices {
		data := v.(map[string]interface{})
		primaries := data["primaries"].(map[string]interface{})

		docs := primaries["docs"].(map[string]interface{})
		count := docs["count"].(float64)
		ch <- prometheus.MustNewConstMetric(c.docsCount, prometheus.GaugeValue, count, index)

		store := primaries["store"].(map[string]interface{})
		size := store["size_in_bytes"].(float64)
		ch <- prometheus.MustNewConstMetric(c.indexSize, prometheus.GaugeValue, size, index)

	}
}
