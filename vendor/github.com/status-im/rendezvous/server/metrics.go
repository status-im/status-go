package server

var (
	metrics MetricsInterface = noopMetrics{}
)

func UseMetrics(m MetricsInterface) {
	metrics = m
}

type MetricsInterface interface {
	AddActiveRegistration(...string)
	RemoveActiveRegistration(...string)
	ObserveDiscoverSize(float64, ...string)
	ObserveDiscoveryDuration(float64, ...string)
	CountError(...string)
}

type noopMetrics struct{}

func (n noopMetrics) AddActiveRegistration(lvs ...string) {}

func (n noopMetrics) RemoveActiveRegistration(lvs ...string) {}

func (n noopMetrics) ObserveDiscoverSize(o float64, lvs ...string) {}

func (n noopMetrics) ObserveDiscoveryDuration(o float64, lvs ...string) {}

func (n noopMetrics) CountError(lvs ...string) {}
