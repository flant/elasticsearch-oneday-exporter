package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type ClusterSettingsCollector struct {
	client *Client
	logger *logrus.Logger

	excludeExists            *prometheus.Desc
	maxShardsPerNode         *prometheus.Desc
	totalShardsPerNodeExists *prometheus.Desc
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
		totalShardsPerNodeExists: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "total_shards_per_node", "exists"),
			"total_shards_per_node exists in cluster settings", labels, constLabels,
		),
		maxShardsPerNode: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "clustersettings_stats", "max_shards_per_node"),
			"Current maximum number of shards per node setting.", labels, constLabels,
		),
	}
}

func (c *ClusterSettingsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.excludeExists
	ch <- c.maxShardsPerNode
	ch <- c.totalShardsPerNodeExists
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

	path = "persistent.cluster..routing.allocation.total_shards_per_node"
	if _, ok := walk(settings, path); ok {
		ch <- prometheus.MustNewConstMetric(c.totalShardsPerNodeExists, prometheus.CounterValue, 1, "persistent")
	} else {
		ch <- prometheus.MustNewConstMetric(c.totalShardsPerNodeExists, prometheus.CounterValue, 0, "persistent")
	}

	path = "transient.cluster.routing.allocation.total_shards_per_node"
	if _, ok := walk(settings, path); ok {
		ch <- prometheus.MustNewConstMetric(c.totalShardsPerNodeExists, prometheus.CounterValue, 1, "transient")
	} else {
		ch <- prometheus.MustNewConstMetric(c.totalShardsPerNodeExists, prometheus.CounterValue, 0, "transient")
	}

	path = "persistent.cluster.max_shards_per_node"
	if count, ok := walk(settings, path); ok {
		if v, ok := count.(string); ok {
			maxShardsPerNode, err := strconv.ParseInt(v, 10, 64)
			if err == nil {
				path_transient := "transient.cluster.max_shards_per_node"
				if count_transient, ok := walk(settings, path_transient); ok {
					if v, ok := count_transient.(string); ok {
						maxShardsPerNodeTransient, err := strconv.ParseInt(v, 10, 64)
						if err == nil {
							ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, float64(maxShardsPerNodeTransient), "transient")
						} else {
							ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, float64(maxShardsPerNode), "persistent")
						}
					} else {
						ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, float64(maxShardsPerNode), "persistent")
					}
				} else {
					ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, float64(maxShardsPerNode), "persistent")
				}
			} else {
				c.logger.Errorf("got invalid %q value: %#v", path, count)
			}
		} else {
			c.logger.Errorf("got invalid %q value: %#v", path, count)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(c.maxShardsPerNode, prometheus.GaugeValue, 1000.0, "default")
	}

}
