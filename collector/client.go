package collector

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/sirupsen/logrus"
)

type Client struct {
	es     *elasticsearch.Client
	logger *logrus.Logger
}

type IndexHealthInfo struct {
	Status           string `json:"status"`
	NumberOfShards   int    `json:"number_of_shards"`
	NumberOfReplicas int    `json:"number_of_replicas"`
}

func NewClient(logger *logrus.Logger, addresses []string, tlsClientConfig *tls.Config) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
		Transport: &http.Transport{
			TLSClientConfig: tlsClientConfig,
		},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{es: es, logger: logger}, nil

}

func (c *Client) GetIndices(s []string) (map[string]interface{}, error) {
	c.logger.Debug("Getting indices stats: ", s)
	resp, err := c.es.Indices.Stats(
		c.es.Indices.Stats.WithIndex(s...),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r["indices"], nil
}

func (c *Client) GetSnapshots(sr string) (map[string][]interface{}, error) {
	c.logger.Debug("Getting snapshots in: ", sr)
	resp, err := c.es.Snapshot.Get(sr, []string{"*"})
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetInfo() (map[string]interface{}, error) {
	c.logger.Debug("Getting cluster info")
	resp, err := c.es.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetMapping(s []string) (map[string]interface{}, error) {
	c.logger.Debug("Getting indices mapping: ", s)
	resp, err := c.es.Indices.GetMapping(
		c.es.Indices.GetMapping.WithIndex(s...),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetSettings(s []string) (map[string]interface{}, error) {
	c.logger.Debug("Getting indices settings: ", s)
	resp, err := c.es.Indices.GetSettings(
		c.es.Indices.GetSettings.WithIndex(s...),
		c.es.Indices.GetSettings.WithIncludeDefaults(true),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetClusterSettings() (map[string]interface{}, error) {
	c.logger.Debug("Getting cluster settings")
	resp, err := c.es.Cluster.GetSettings(
		c.es.Cluster.GetSettings.WithIncludeDefaults(false),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %s", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("request failed: %v", resp.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) GetIndicesHealth(indices []string) (map[string]IndexHealthInfo, error) {
	resp, err := c.es.Cluster.Health(
		c.es.Cluster.Health.WithIndex(indices...),
		c.es.Cluster.Health.WithLevel("indices"),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster health: %s", err)
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return nil, fmt.Errorf("health request failed: %s", resp.String())
	}
	var body struct {
		Indices map[string]IndexHealthInfo `json:"indices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Indices, nil
}
