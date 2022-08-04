package main

import (
	"crypto/tls"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	indexGroupLastTotalBytes = make(map[string]float64)
)

type Collector struct {
	client *Client

	indexSize      *prometheus.Desc
	indexGroupSize *prometheus.Desc
	docsCount      *prometheus.Desc
	snapshotsCount *prometheus.Desc
}

func NewCollector(address, project string, repo string, tlsClientConfig *tls.Config) (*Collector, error) {
	namespace := "oneday_elasticsearch"
	labels := []string{"index", "index_group"}
	slabels := []string{"repository"}
	labels_group := []string{"index_group"}

	client, err := NewClient([]string{address}, tlsClientConfig)
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
		indexGroupSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_group_store", "size_bytes"),
			"Size of each index to date", labels_group, constLabels,
		),
		docsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_docs", "total"),
			"Count of docs for each index to date", labels, constLabels,
		),
		snapshotsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots_count", "total"),
			"Count of snapshots", slabels, constLabels,
		),
	}, nil
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.indexSize
	ch <- c.indexGroupSize
	ch <- c.docsCount
	ch <- c.snapshotsCount
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	indices, err := c.client.GetIndices([]string{fmt.Sprintf("*-%s", time.Now().Format("2006.01.02"))})
	if err != nil {
		log.Fatal("error getting indices stats: ", err)
	}

	indexGroupSize := make(map[string]float64)
	for index, v := range indices {
		// Find date -Y.m.d (-2021.12.01) and replace
		reFindDateTime := regexp.MustCompile(`-\d+.\d+.\d+$`)
		// Create variable with index prefix
		indexGrouplabel := strings.ToLower(reFindDateTime.ReplaceAllString(index, ""))

		data := v.(map[string]interface{})
		primaries := data["primaries"].(map[string]interface{})

		if primaries["indexing"] == nil {
			continue
		}

		docs := primaries["indexing"].(map[string]interface{})
		count := docs["index_total"].(float64)
		ch <- prometheus.MustNewConstMetric(c.docsCount, prometheus.GaugeValue, count, index, indexGrouplabel)

		store := primaries["store"].(map[string]interface{})
		size := store["size_in_bytes"].(float64)
		ch <- prometheus.MustNewConstMetric(c.indexSize, prometheus.GaugeValue, size, index, indexGrouplabel)

		var lastIndexDifferenceBytes float64
		if _, ok := indexGroupLastTotalBytes[index]; ok {
			if size > indexGroupLastTotalBytes[index] {
				lastIndexDifferenceBytes = size - indexGroupLastTotalBytes[index]
			} else {
				lastIndexDifferenceBytes = 0
			}
		}

		indexGroupLastTotalBytes[index] = size
		indexGroupSize[indexGrouplabel] += lastIndexDifferenceBytes
	}

	for indexGroup, v := range indexGroupSize {
		ch <- prometheus.MustNewConstMetric(c.indexGroupSize, prometheus.CounterValue, v, indexGroup)
	}
	if *repoName != "" {
		snapshots, err := c.client.GetSnapshots(*repoName)
		if err != nil {
			log.Fatal("error getting snapshots count: ", err)
		}
		s := len(snapshots["snapshots"])
		ch <- prometheus.MustNewConstMetric(c.snapshotsCount, prometheus.GaugeValue, float64(s), *repoName)
	}
}
