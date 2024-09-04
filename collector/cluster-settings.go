package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type ClusterSettingsCollector struct {
	client *Client
	logger *logrus.Logger

	excludeExists    *prometheus.Desc
	maxShardsPerNode *prometheus.Desc
}

func NewClusterSettingsCollector(logger *logrus.Logger, client *Client, labels, labels_group []string, datepattern string,
	constLabels prometheus.Labels) *ClusterSettingsCollector {

	return &ClusterSettingsCollector{
		client: client,
		logger: logger,
		excludeExists: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "exclude", "exists"),
			"Exclude exists in cluster settings", labels, constLabels,
		),
		maxShardsPerNode: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clustersettings_stats", "max_shards_per_node"),
			"Current maximum number of shards per node setting.",
			nil, nil,
		),
	}
}

func (c *ClusterSettingsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.excludeExists
	ch <- c.maxShardsPerNode
}

func (c *ClusterSettingsCollector) Collect(ch chan<- prometheus.Metric) {
	settings, err := c.client.GetClusterSettings()
	if err != nil {
		c.logger.Fatalf("error getting indices settings: %v", err)
	}

	path := "persistent.cluster.routing.allocation.exclude"
	if _, ok := walk(settings, path); ok {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "persistent")
	} else {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "persistent")
	}

	path = "transient.cluster.routing.allocation.exclude"
	if _, ok := walk(settings, path); ok {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "transient")
	} else {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "transient")
	}

	path = "persistent.cluster.max_shards_per_node"
	if count, ok := walk(settings, path); ok {
		if v, ok := count.(string); ok {
			maxShardsPerNode, err := strconv.ParseInt(v, 10, 64)
			if err == nil {
				ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, float64(maxShardsPerNode))
			} else {
				c.logger.Errorf("got invalid %q value: %#v", path, count)
			}
		} else {
			c.logger.Errorf("got invalid %q value: %#v", path, count)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, 1000.0)
	}

}
