package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type SettingsCollector struct {
	client *Client
	logger *logrus.Logger

	fieldsLimit      *prometheus.Desc
	fieldsGroupLimit *prometheus.Desc
}

func NewSettingsCollector(logger *logrus.Logger, client *Client, labels, labels_group []string,
	constLabels prometheus.Labels) *SettingsCollector {

	return &SettingsCollector{
		client: client,
		logger: logger,
		fieldsLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_limit", "total"),
			"Total limit of fields of each index to date", labels, constLabels,
		),
		fieldsGroupLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_group_limit", "total"),
			"Total limit of fields of each index group to date", labels_group, constLabels,
		),
	}
}

func (c *SettingsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fieldsLimit
	ch <- c.fieldsGroupLimit
}

func (c *SettingsCollector) Collect(ch chan<- prometheus.Metric) {
	today := todayFunc()
	indicesPattern := indicesPatternFunc(today)

	settings, err := c.client.GetSettings([]string{indicesPattern})
	if err != nil {
		c.logger.Fatalf("error getting indices settings: %v", err)
	}

	fieldsGroupLimit := make(map[string]float64)
	for index, v := range settings {
		// Create variable with index prefix
		indexGrouplabel := indexGroupLabelFunc(index, today)

		data, ok := v.(map[string]interface{})
		if !ok {
			c.logger.Error("got invalid index setttings for: %s", index)
			continue
		}

		path := "index.mapping.total_fields.limit"
		limit, ok := walk(data, "settings."+path)
		if !ok {
			limit, ok = walk(data, "defaults."+path)
			if !ok {
				c.logger.Errorf("%q was not found for: %s", path, index)
				continue
			}
		}

		if v, ok := limit.(float64); ok {
			ch <- prometheus.MustNewConstMetric(c.fieldsLimit, prometheus.GaugeValue, v, index, indexGrouplabel)
			fieldsGroupLimit[indexGrouplabel] += v
		} else {
			c.logger.Error("got invalid %q value for: %s", path, index)
		}
	}

	for indexGroup, v := range fieldsGroupLimit {
		ch <- prometheus.MustNewConstMetric(c.fieldsGroupLimit, prometheus.GaugeValue, v, indexGroup)
	}
}
