package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "nais"
const subsystem = "certificator"

const labelErrorCode = "error_code"

var (
	namespaces = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "namespaces_total",
		Help:      "Number of namespaces Certificator saves CA bundles into.",
	})

	pendingNamespaces = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "namespaces_pending",
		Help:      "Number of namespaces that are lacking the latest CA bundle updates.",
	})

	certificates = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "certificates",
		Help:      "Number of CA certificates in the bundle.",
	})

	sync = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "synchronizations",
		Help:      "Indicates how many Kubernetes synchronizations are attempted.",
	}, []string{labelErrorCode})

	refresh = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "downloads",
		Help:      "Indicates how many certificate refreshes attempted.",
	}, []string{labelErrorCode})
)

func init() {
	prometheus.MustRegister(
		namespaces,
		pendingNamespaces,
		certificates,
		sync,
		refresh,
	)

	namespaces.Set(0)
	pendingNamespaces.Set(0)
	certificates.Set(0)
	sync.WithLabelValues("0")
	sync.WithLabelValues("1")
	refresh.WithLabelValues("0")
	refresh.WithLabelValues("1")
}

func SetTotalNamespaces(count int) {
	namespaces.Set(float64(count))
}

func SetPendingNamespaces(count int) {
	pendingNamespaces.Set(float64(count))
}

func SetCertificates(count int) {
	certificates.Set(float64(count))
}

func IncSync(errorCode int) {
	sync.WithLabelValues(strconv.Itoa(errorCode)).Inc()
}

func IncRefresh(errorCode int) {
	refresh.WithLabelValues(strconv.Itoa(errorCode)).Inc()
}
