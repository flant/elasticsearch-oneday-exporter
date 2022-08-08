package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type FieldsCollector struct {
	client *Client
	logger *logrus.Logger

	fieldsCount      *prometheus.Desc
	fieldsGroupCount *prometheus.Desc
}

func NewFieldsCollector(logger *logrus.Logger, client *Client, labels, labels_group []string,
	constLabels prometheus.Labels) *FieldsCollector {

	return &FieldsCollector{
		client: client,
		logger: logger,
		fieldsCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_count", "total"),
			"Count of fields of each index to date", labels, constLabels,
		),
		fieldsGroupCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_group_count", "total"),
			"Total number of fields of each index group to date", labels_group, constLabels,
		),
	}
}

func (c *FieldsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fieldsCount
	ch <- c.fieldsGroupCount
}

func (c *FieldsCollector) Collect(ch chan<- prometheus.Metric) {
	today := todayFunc()
	indicesPattern := indicesPatternFunc(today)

	mapping, err := c.client.GetMapping([]string{indicesPattern})
	if err != nil {
		c.logger.Fatalf("error getting indices mapping: %v", err)
	}

	fieldsGroupCount := make(map[string]float64)
	for index, v := range mapping {
		// Create variable with index prefix
		indexGrouplabel := indexGroupLabelFunc(index, today)

		data, ok := v.(map[string]interface{})
		if !ok {
			c.logger.Errorf("got invalid mapping for: %s", index)
		}

		count := countFields(data, 0)

		ch <- prometheus.MustNewConstMetric(c.fieldsCount, prometheus.GaugeValue, count, index, indexGrouplabel)

		fieldsGroupCount[indexGrouplabel] += count
	}

	for indexGroup, v := range fieldsGroupCount {
		ch <- prometheus.MustNewConstMetric(c.fieldsGroupCount, prometheus.GaugeValue, v, indexGroup)
	}
}
