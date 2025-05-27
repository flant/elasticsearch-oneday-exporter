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

	indexSize      *prometheus.Desc
	indexTotalSize *prometheus.Desc
	indexGroupSize *prometheus.Desc
	docsCount      *prometheus.Desc
	shardsDocs     *prometheus.Desc
	indexHealth    *prometheus.Desc
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
	}
}

func (c *IndicesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.indexSize
	ch <- c.indexTotalSize
	ch <- c.docsCount
	ch <- c.indexGroupSize
	ch <- c.shardsDocs
	ch <- c.indexHealth
}

func (c *IndicesCollector) Collect(ch chan<- prometheus.Metric) {
	today := todayFunc(c.datePattern)
	indicesPattern := indicesPatternFunc(today)

	indices, err := c.client.GetIndices([]string{indicesPattern})
	if err != nil {
		c.logger.Fatalf("error getting indices stats: %v", err)
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

	indexGroupSize := make(map[string]float64, len(indices))
	for index, v := range indices {
		// Create variable with index prefix
		indexGrouplabel := indexGroupLabelFunc(index, today)

		data, ok := v.(map[string]interface{})
		if !ok {
			c.logger.Errorf("got invalid index stats for: %s", index)
			continue
		}

		path := "primaries.indexing.index_total"
		if count, ok := walk(data, path); ok {
			if v, ok := count.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.docsCount, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, count)
			}
		} else {
			c.logger.Errorf("%q was not found for: %s", path, index)
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
		} else {
			c.logger.Errorf("%q was not found for: %s", path, index)
		}

		path = "total.store.size_in_bytes"
		if size, ok := walk(data, path); ok {
			if v, ok := size.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.indexTotalSize, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, size)
			}
		} else {
			c.logger.Errorf("%q was not found for: %s", path, index)
		}

		path = "primaries.docs.count"
		if count, ok := walk(data, path); ok {
			if v, ok := count.(float64); ok {
				ch <- prometheus.MustNewConstMetric(c.shardsDocs, prometheus.GaugeValue, v, index, indexGrouplabel)
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path, index, count)
			}
		} else {
			c.logger.Errorf("%q was not found for: %s", path, index)
		}

	}

	for indexGroup, v := range indexGroupSize {
		ch <- prometheus.MustNewConstMetric(c.indexGroupSize, prometheus.CounterValue, v, indexGroup)
	}
}
