package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	BookingsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "seki_bookings_created_total",
		Help: "Total number of successfully created bookings.",
	})

	BookingConflicts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "seki_booking_conflicts_total",
		Help: "Total number of booking attempts rejected due to an overlapping slot (proves the exclusion constraint is doing its job under load).",
	})

	BookingsCancelled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "seki_bookings_cancelled_total",
		Help: "Total number of cancelled bookings.",
	})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "seki_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
)
