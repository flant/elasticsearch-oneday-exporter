package collector

import (
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
	settingsExclude, err := c.client.GetClusterSettingsExclude()
	if err != nil {
		c.logger.Fatalf("error getting indices settings with exclude section: %v", err)
	}

	if len(settingsExclude) == 0 {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "persistent")
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "transient")
	} else {
		if settingsExclude["persistent"] == nil {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "persistent")
		} else {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "persistent")
		}
		if settingsExclude["transient"] == nil {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "transient")
		} else {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "transient")
		}
	}
	settings, err := c.client.GetClusterSettings()
	if err != nil {
		c.logger.Fatalf("error getting indices settings: %v", err)
	}

	path := "persistent.cluster.max_shards_per_node"
	if count, ok := walk(settings, path); ok {
		if v, ok := count.(float64); ok {
			ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, v)
		} else {
			c.logger.Errorf("got invalid %q value: %#v", path, count)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, 1000.0)
	}

}
