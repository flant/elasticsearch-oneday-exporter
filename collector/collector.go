package collector

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	namespace = "oneday_elasticsearch"
)

var (
	labels       = []string{"index", "index_group"}
	labels_group = []string{"index_group"}
	slabels      = []string{"repository"}
)

func NewCollector(logger *logrus.Logger, address, project string, repo string, tlsClientConfig *tls.Config) error {
	client, err := NewClient(logger, []string{address}, tlsClientConfig)
	if err != nil {
		return fmt.Errorf("error creating the client: %v", err)
	}

	info, err := client.GetInfo()
	if err != nil {
		return fmt.Errorf("error getting cluster info: %v", err)
	}
	logger.Infof("Cluster info: %v", info)

	cluster := info["cluster_name"].(string)

	constLabels := prometheus.Labels{
		"cluster": cluster,
		"project": project,
	}

	err = prometheus.Register(NewFieldsCollector(logger, client, labels, labels_group, constLabels))
	if err != nil {
		return fmt.Errorf("error registering index fields count collector: %v", err)
	}

	err = prometheus.Register(NewIndicesCollector(logger, client, labels, labels_group, constLabels))
	if err != nil {
		return fmt.Errorf("error registering indices stats collector: %v", err)
	}

	err = prometheus.Register(NewSettingsCollector(logger, client, labels, labels_group, constLabels))
	if err != nil {
		return fmt.Errorf("error registering indices settings collector: %v", err)
	}

	if repo != "" {
		err = prometheus.Register(NewSnapshotCollector(logger, client, repo, slabels, constLabels))
		if err != nil {
			return fmt.Errorf("error registering snapshots stats collector: %v", err)
		}
	}

	return nil
}

func todayFunc() string { return time.Now().Format("2006.01.02") }

func indicesPatternFunc(today string) string { return fmt.Sprintf("*-%s", today) }

// Find date -Y.m.d (-2021.12.01) and replace
func indexGroupLabelFunc(index, today string) string {
	return strings.ToLower(strings.TrimSuffix(index, "-"+today))
}

// Walk over the json map by path `f1.f2.f3` and return the last field
func walk(m map[string]interface{}, path string) (interface{}, bool) {
	p := strings.Split(path, ".")
	for _, v := range p[:len(p)-1] {
		if v, ok := m[v]; ok {
			switch v := v.(type) {
			case map[string]interface{}:
				m = v
			default:
				return v, false
			}
		}
	}

	v, ok := m[p[len(p)-1]]

	return v, ok
}

// Count mapping fields by `type` field
// https://github.com/elastic/elasticsearch/issues/68947#issue-806860754
func countFields(m map[string]interface{}, count float64) float64 {
	if v, ok := m["type"]; ok {
		switch v := v.(type) {
		case map[string]interface{}:
			count = countFields(v, count)
		case string:
			return count + 1
		}
	}

	for _, v := range m {
		switch v := v.(type) {
		case map[string]interface{}:
			count = countFields(v, count)
		}
	}

	return count
}
