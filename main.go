package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Same struct prometheus uses for their /version address.
// Separate copy to avoid pulling all of prometheus as a dependency
type prometheusVersion struct {
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Branch    string `json:"branch"`
	BuildUser string `json:"buildUser"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

var (
	log      = logrus.New()
	logLevel = kingpin.Flag("log.level",
		"Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]",
	).Default("info").Enum("debug", "info", "warn", "error", "fatal")
	logFormat = kingpin.Flag("log.format",
		"Set the log format. Valid formats: [json, text]",
	).Default("json").Enum("json", "text")

	listenAddress = kingpin.Flag("telemetry.addr", "Listen on host:port.").
			Default(":9141").String()
	metricsPath = kingpin.Flag("telemetry.path", "URL path for surfacing collected metrics.").
			Default("/metrics").String()

	address = kingpin.Flag("address", "Elasticsearch node to use.").
		Default("http://localhost:9200").String()
	insecure = kingpin.Flag("insecure", "Allow insecure server connections when using SSL.").
			Default("false").Bool()

	projectName = kingpin.Flag("project", "Project name").String()
)

func main() {
	kingpin.Version(version.Print("es-oneday-exporter"))
	kingpin.HelpFlag.Short('h')

	kingpin.Parse()

	if err := setLogLevel(*logLevel); err != nil {
		log.Fatal(err)
	}
	if err := setLogFormat(*logFormat); err != nil {
		log.Fatal(err)
	}

	collector, err := NewCollector(*address, *projectName, *insecure)
	if err != nil {
		log.Fatal("error creating new collector instance: ", err)
	}

	prometheus.MustRegister(collector)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/healthz", healthCheck)
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		// we can't use "version" directly as it is a package, and not an object that
		// can be serialized.
		err := json.NewEncoder(w).Encode(prometheusVersion{
			Version:   version.Version,
			Revision:  version.Revision,
			Branch:    version.Branch,
			BuildUser: version.BuildUser,
			BuildDate: version.BuildDate,
			GoVersion: version.GoVersion,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err), http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
<head><title>es-oneday-exporter</title></head>
<body>
<h1>es-oneday-exporter</h1>
<p><a href="` + *metricsPath + `">Metrics</a></p>
<p><i>` + version.Info() + `</i></p>
</body>
</html>`))
	})

	log.Info("Starting es-oneday-exporter", version.Info())
	log.Info("Build context", version.BuildContext())

	log.Info("Starting server on ", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintln(w, `{"status":"ok"}`)
	if err != nil {
		log.Debugf("Failed to write to stream: %v", err)
	}
}

func setLogLevel(level string) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(lvl)

	return nil
}

func setLogFormat(format string) error {
	var formatter logrus.Formatter

	switch format {
	case "text":
		formatter = &logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		}
	case "json":
		formatter = &logrus.JSONFormatter{}
	default:
		return fmt.Errorf("invalid log format: %s", format)
	}

	log.SetFormatter(formatter)

	return nil
}
