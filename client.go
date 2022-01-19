package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v7"
)

type Client struct {
	es *elasticsearch.Client
}

func NewClient(addresses []string, insecure bool) (*Client, error) {
	cfg := elasticsearch.Config{
		Addresses: addresses,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{es}, nil

}

func (c *Client) GetIndices(s []string) (map[string]interface{}, error) {
	log.Debug("Getting indices stats: ", s)
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

func (c *Client) GetInfo() (map[string]interface{}, error) {
	log.Debug("Getting cluster info")
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

func (c *Client) RefreshIndices() error {
	log.Debug("refresh indices")
	resp, err := c.es.Indices.Refresh(
		c.es.Indices.Refresh.WithIndex([]string{fmt.Sprintf("*-%s", time.Now().Format("2006.01.02"))}...),
	)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("error getting indices stats: ", err)
	}
	return nil

}
