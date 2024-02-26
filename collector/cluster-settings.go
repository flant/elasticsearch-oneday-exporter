package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type ClusterSettingsCollector struct {
	client *Client
	logger *logrus.Logger

	excludeExists *prometheus.Desc
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
	}
}

func (c *ClusterSettingsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.excludeExists
}

func (c *ClusterSettingsCollector) Collect(ch chan<- prometheus.Metric) {
	settings, err := c.client.GetClusterSettings()
	if err != nil {
		c.logger.Fatalf("error getting indices settings: %v", err)
	}

	if len(settings) == 0 {
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "persistent")
		ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "transient")
	} else {
		if settings["persistent"] == nil {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "persistent")
		} else {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "persistent")
		}
		if settings["transient"] == nil {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 0, "transient")
		} else {
			ch <- prometheus.MustNewConstMetric(c.excludeExists, prometheus.CounterValue, 1, "transient")
		}
	}
}
