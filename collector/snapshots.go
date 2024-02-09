package collector

import (
	"github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
)

type SnapshotCollector struct {
	client *Client
	logger *logrus.Logger

	repo string

	snapshotsCount *prometheus.Desc
}

func NewSnapshotCollector(logger *logrus.Logger, client *Client, repo string, labels []string,
	constLabels prometheus.Labels) *SnapshotCollector {

	return &SnapshotCollector{
		client: client,
		logger: logger,
		repo:   repo,
		snapshotsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "snapshots_count", "total"),
			"Count of snapshots", slabels, constLabels,
		),
	}
}

func (c *SnapshotCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.snapshotsCount
}

func (c *SnapshotCollector) Collect(ch chan<- prometheus.Metric) {
	snapshots, err := c.client.GetSnapshots(c.repo)
	if err != nil {
		c.logger.Fatalf("error getting snapshots count: %v", err)
	}
	s := len(snapshots["snapshots"])
	ch <- prometheus.MustNewConstMetric(c.snapshotsCount, prometheus.GaugeValue, float64(s), c.repo)
}
