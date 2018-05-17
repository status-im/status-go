package main

import "net/http"

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<html>
			<head><title>Status Node Exporter</title></head>
			<body>
			<h1>Status Node Exporter</h1>
			<p><a href="` + metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
}

func metricsHandler(c *collector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := c.collect()
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(err.Error()))
			return
		}

		w.Write([]byte(body))
	}
}
