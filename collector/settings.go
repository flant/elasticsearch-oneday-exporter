package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type SettingsCollector struct {
	client *Client
	logger *logrus.Logger

	datePattern string

	fieldsLimit         *prometheus.Desc
	fieldsGroupLimit    *prometheus.Desc
	readOnlyAllowDelete *prometheus.Desc
	readOnly            *prometheus.Desc
}

func NewSettingsCollector(logger *logrus.Logger, client *Client, labels, labels_group []string, datepattern string,
	constLabels prometheus.Labels) *SettingsCollector {

	return &SettingsCollector{
		client:      client,
		logger:      logger,
		datePattern: datepattern,
		fieldsLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_limit", "total"),
			"Total limit of fields of each index to date", labels, constLabels,
		),
		fieldsGroupLimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fields_group_limit", "total"),
			"Total limit of fields of each index group to date", labels_group, constLabels,
		),
		readOnlyAllowDelete: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "read_only_allow_delete", "total"),
			"State of the read_only_allow_delete field, which means that the index is in readonly mode", labels, constLabels,
		),
		readOnly: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "read_only", "total"),
			"State of the read_only field, which means that the index is in readonly mode", labels, constLabels,
		),
	}
}

func (c *SettingsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fieldsLimit
	ch <- c.fieldsGroupLimit
	ch <- c.readOnlyAllowDelete
	ch <- c.readOnly
}

func (c *SettingsCollector) Collect(ch chan<- prometheus.Metric) {
	today := todayFunc(c.datePattern)
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

		path_limit := "index.mapping.total_fields.limit"
		limit, ok := walk(data, "settings."+path_limit)
		if !ok {
			limit, ok = walk(data, "defaults."+path_limit)
			if !ok {
				c.logger.Errorf("%q was not found for: %s", path_limit, index)
			}
		} else {
			if s, ok := limit.(string); ok {
				if v, err := strconv.ParseFloat(s, 64); err == nil {
					ch <- prometheus.MustNewConstMetric(c.fieldsLimit, prometheus.GaugeValue, v, index, indexGrouplabel)
					fieldsGroupLimit[indexGrouplabel] += v
				} else {
					c.logger.Errorf("error parsing %q value for: %s: %v ", path_limit, index, err)
				}
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path_limit, index, limit)
			}
		}

		path_block := "index.blocks.read_only_allow_delete"
		block, ok := walk(data, "settings."+path_block)
		if !ok {
			ch <- prometheus.MustNewConstMetric(c.readOnlyAllowDelete, prometheus.GaugeValue, 0, index, indexGrouplabel)
		} else {
			if s, ok := block.(string); ok {
				if v, err := strconv.ParseBool(s); err == nil {
					if v {
						ch <- prometheus.MustNewConstMetric(c.readOnlyAllowDelete, prometheus.GaugeValue, 1, index, indexGrouplabel)
					} else {
						ch <- prometheus.MustNewConstMetric(c.readOnlyAllowDelete, prometheus.GaugeValue, 0, index, indexGrouplabel)
					}
				} else {
					c.logger.Errorf("error parsing %q value for: %s: %v ", path_block, index, err)
				}
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path_block, index, limit)
			}
		}

		path_roblock := "index.blocks.read_only"
		roblock, ok := walk(data, "settings."+path_roblock)
		if !ok {
			ch <- prometheus.MustNewConstMetric(c.readOnly, prometheus.GaugeValue, 0, index, indexGrouplabel)
		} else {
			if s, ok := roblock.(string); ok {
				if v, err := strconv.ParseBool(s); err == nil {
					if v {
						ch <- prometheus.MustNewConstMetric(c.readOnly, prometheus.GaugeValue, 1, index, indexGrouplabel)
					} else {
						ch <- prometheus.MustNewConstMetric(c.readOnly, prometheus.GaugeValue, 0, index, indexGrouplabel)
					}
				} else {
					c.logger.Errorf("error parsing %q value for: %s: %v ", path_roblock, index, err)
				}
			} else {
				c.logger.Errorf("got invalid %q value for: %s value: %#v", path_roblock, index, limit)
			}
		}
	}

	for indexGroup, v := range fieldsGroupLimit {
		ch <- prometheus.MustNewConstMetric(c.fieldsGroupLimit, prometheus.GaugeValue, v, indexGroup)
	}
}
