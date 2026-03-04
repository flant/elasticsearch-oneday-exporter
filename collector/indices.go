package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	indexGroupLastTotalBytes = make(map[string]float64)
)

type IndicesCollector struct {
	client *Client
	logger *logrus.Logger

	datePattern string

	indexSize             *prometheus.Desc
	indexTotalSize        *prometheus.Desc
	indexGroupSize        *prometheus.Desc
	docsCount             *prometheus.Desc
	shardsDocs            *prometheus.Desc
	indexHealth           *prometheus.Desc
	indexStatsUnavailable *prometheus.Desc
}

func NewIndicesCollector(logger *logrus.Logger, client *Client, labels, labels_group []string, labels_health []string, datepattern string,
	constLabels prometheus.Labels) *IndicesCollector {

	return &IndicesCollector{
		client:      client,
		logger:      logger,
		datePattern: datepattern,
		indexSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_store", "size_bytes_primary"),
			"Size of each index to date", labels, constLabels,
		),
		indexTotalSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_store", "size_bytes_total"),
			"Total (primary + all replicas) size of each index to date", labels, constLabels,
		),
		docsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_docs", "total"),
			"Count of docs for each index to date", labels, constLabels,
		),
		indexGroupSize: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_group_store", "size_bytes"),
			"Total size of each index group to date", labels_group, constLabels,
		),
		shardsDocs: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices", "shards_docs"),
			"Count of documents on this shard", labels, constLabels,
		),
		indexHealth: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices_health", "status"),
			"Health status of each index: green=0,yellow=1,red=2", labels_health, constLabels,
		),
		indexStatsUnavailable: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "indices", "stats_unavailable"),
			"1 if index stats were skipped (e.g. ES bug with negative longs); use for alerting on problematic indices", labels, constLabels,
		),
	}
}

func (c *IndicesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.indexSize
	ch <- c.indexTotalSize
	ch <- c.docsCount
	ch <- c.indexGroupSize
	ch <- c.shardsDocs
	ch <- c.indexHealth
	ch <- c.indexStatsUnavailable
}

func (c *IndicesCollector) Collect(ch chan<- prometheus.Metric) {
	today := todayFunc(c.datePattern)
	indicesPattern := indicesPatternFunc(today)

	indices, err := c.client.GetIndices([]string{indicesPattern})
	if err != nil {
		c.logger.Errorf("error getting indices stats: %v", err)
		return
	}

	healthMap, err := c.client.GetIndicesHealth([]string{"*"})
	if err != nil {
		c.logger.Errorf("error getting indices health: %v", err)
	} else {
		for idx, info := range healthMap {
			var val float64
			switch info.Status {
			case "green":
				val = 0
			case "yellow":
				val = 1
			case "red":
				val = 2
			default:
				continue
			}
			ch <- prometheus.MustNewConstMetric(
				c.indexHealth,
				prometheus.GaugeValue,
				val,
				idx,
				strconv.Itoa(info.NumberOfReplicas),
			)
		}
	}

	requiredPaths := []string{
		"primaries.indexing.index_total",
		"primaries.store.size_in_bytes",
		"primaries.docs.count",
		"total.store.size_in_bytes",
	}

	indexGroupSize := make(map[string]float64, len(indices))
	for index, v := range indices {
		indexGrouplabel := indexGroupLabelFunc(index, today)

		data, ok := v.(map[string]interface{})
		if !ok {
			c.logger.Errorf("got invalid index stats for: %s", index)
			continue
		}

		statsComplete := true
		for _, path := range requiredPaths {
			if _, ok := walk(data, path); !ok {
				statsComplete = false
				c.logger.Warnf("skipping index %s: required stats incomplete (missing %q), likely OS negative longs bug", index, path)
				break
			}
		}
		if !statsComplete {
			ch <- prometheus.MustNewConstMetric(c.indexStatsUnavailable, prometheus.GaugeValue, 1, index, indexGrouplabel)
			continue
		}

		path := "primaries.indexing.index_total"
		if count, ok := walk(data, path); ok {
			if v, ok := count.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.docsCount, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, count)
			}
		}

		path = "primaries.store.size_in_bytes"
		if size, ok := walk(data, path); ok {
			if v, ok := size.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.indexSize, prometheus.GaugeValue, v, index, indexGrouplabel)

				var lastIndexDifferenceBytes float64 = 0
				if _, ok := indexGroupLastTotalBytes[index]; ok {
					if v > indexGroupLastTotalBytes[index] {
						lastIndexDifferenceBytes = v - indexGroupLastTotalBytes[index]
					}
				}
				indexGroupLastTotalBytes[index] = v
				indexGroupSize[indexGrouplabel] += lastIndexDifferenceBytes
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, size)
			}
		}

		path = "total.store.size_in_bytes"
		if size, ok := walk(data, path); ok {
			if v, ok := size.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.indexTotalSize, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, size)
			}
		}

		path = "primaries.docs.count"
		if count, ok := walk(data, path); ok {
			if v, ok := count.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.shardsDocs, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, count)
			}
		}
	}

	for indexGroup, v := range indexGroupSize {
		ch <- prometheus.MustNewConstMetric(c.indexGroupSize, prometheus.CounterValue, v, indexGroup)
	}
}
