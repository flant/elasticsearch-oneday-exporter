package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"regexp"
	"time"
	"log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"
)

const namespace = "oneday_elasticsearch"

var (
	pattern_digital = "^[[:digit:]]"
	clusterName = kingpin.Flag("cluster", "Cluster's name").Default("opendistro").String()
	re = regexp.MustCompile("[0-9]+")
	data []Index
	currentDate = time.Now()
	metricsPath = kingpin.Flag("path", "URL path for collected metrics").Default("/metrics").String()
	listenPort = kingpin.Flag("port", "Export's port").Default(":9101").String()
	esUrl = kingpin.Flag("esUrl", "ElasticSearch's URL").Default("https://opendistro").String()
	esPort = kingpin.Flag("esPort", "ElasticSearch's port").Default("9200").String()
	label = []string{"index", "cluster"}
	indexSize = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "indices_store", "size_bytes_primary"), "Size for each index in today", label, nil,
	)
)

type Index struct {
	Name    string `json:"index"`
	Docs 	int `json:"dc"`
	Size    string `json:"pri.store.size"`
}

func match(pattern string, text string) string {
    matched, _ := regexp.Match(pattern, []byte(text))
	var number = "digital"
	var str = "string"
  if matched {
        return number
    } else {
        return str
    }
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintln(w, `{"status":"ok"}`)
	if err != nil {
		log.Println("Failed to write to stream: %v", err)
	}
}

type Collector struct{
}

func (i *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- indexSize
}

func (i *Collector) Collect(ch chan<- prometheus.Metric) {

	cUrl := fmt.Sprintf("%s:%s/_cat/indices/*-%s?format=json&pretty&h=index,dc,pri.store.size", *esUrl, *esPort , currentDate.Format("2006.01.02"))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", cUrl, nil)
	if err != nil {
		log.Fatal("Error! Bad request.")
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error! Query doesn't return a result.")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(body), &data)
	for _, value := range data {
		get_symbol := value.Size[len(value.Size)-2:]
		check_symbol := match(pattern_digital, get_symbol)
		switch check_symbol {
		case "digital":
			originSize, _ := strconv.Atoi(re.FindAllString(value.Size, -1)[0])
			ch <- prometheus.MustNewConstMetric(
				indexSize, prometheus.GaugeValue, float64(originSize), value.Name, *clusterName,
			)
		case "string":
			measure := value.Size[len(value.Size)-2:]
			switch measure {
			case "gb":
				originSize, _ := strconv.Atoi(re.FindAllString(value.Size, -1)[0])
				newSize := originSize * 1024 * 1024 * 1024
				ch <- prometheus.MustNewConstMetric(
					indexSize, prometheus.GaugeValue, float64(newSize), value.Name, *clusterName,
				)
			case "mb":
				originSize, _ := strconv.Atoi(re.FindAllString(value.Size, -1)[0])
				newSize := originSize * 1024 * 1024
				ch <- prometheus.MustNewConstMetric(
					indexSize, prometheus.GaugeValue, float64(newSize), value.Name, *clusterName,
				)
			case "kb":
				originSize, _ := strconv.Atoi(re.FindAllString(value.Size, -1)[0])
				newSize := originSize * 1024
				ch <- prometheus.MustNewConstMetric(
					indexSize, prometheus.GaugeValue, float64(newSize), value.Name, *clusterName,
				)
			}
		}
	}
}

func main(){
	kingpin.Version("0.0.1")
	kingpin.Parse()

	prometheus.MustRegister(&Collector{})

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/healthz", healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
<head><title>oneday-exporter</title></head>
<body>
<h1>oneday-exporter</h1>
<p><a href="` + *metricsPath + `">Metrics</a></p>
</body>
</html>`))
	})

	log.Println("Starting oneday-exporter")
	log.Println("Starting server on", *listenPort)
	log.Fatal(http.ListenAndServe(*listenPort, nil))
}