# Status Metrics

This repository allows running a Dockerized instance of Prometheus and Grafana with a dashboard for collecting Waku metrics from a running instance of status-go.

## Usage

### Run Status Desktop

This has been tested against `status-desktop`. To enable the Waku metrics port, you must run the application with the `metrics` flag.

On macOS, this can be done by running the following command:

```bash
/Applications/Status.app/Contents/MacOS/nim_status_client --metrics
```

If you want to run from source, run following make command:

```bash
make -j10 run ARGS="--datadir=/Users/<your-user>/status-desktop/tmp/app-data --metrics"
```

By default, it uses host and port `0.0.0.0:9305`. You can set the port using `--metrics-address 0.0.0.0:9305`. Make sure the same port is set in [`prometheus/prometheus.yml`](prometheus/prometheus.yml).

### Enable Telemetry
Once logged in, make sure that **Telemetry** is switched on in the advanced settings of the application, **restart** the app.


### Run Status Metrics

Start the local prometheus instance and grafana dashboard by running following command in `status-metrics`:

```bash
docker-compose up -d
```

### Access Grafana

You can now access Grafana at `http://localhost:3000`. Login with the default username `admin` and password `admin`. 

An existing dashboard is available at `http://localhost:3000/d/status-go-metrics/status-go-metrics?orgId=1&from=now-5m&to=now&refresh=5s`. Any changes to the dashboard can be saved by copying the dashboard JSON and overwriting the file in [`grafana/provisioning/dashboards/status-go.json`](grafana/provisioning/dashboards/status-go.json).
