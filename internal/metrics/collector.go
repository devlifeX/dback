package metrics

import (
	"context"
	"net/http"

	"dback/internal/event"
	"dback/internal/operation"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Collector struct {
	opsTotal      *prometheus.CounterVec
	opsActive     prometheus.Gauge
	tasksSkipped  prometheus.Counter
	notifyFailed  prometheus.Counter
	reg           *prometheus.Registry
	unsubs        []func()
}

func NewCollector() *Collector {
	reg := prometheus.NewRegistry()
	c := &Collector{
		reg: reg,
		opsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dback_operations_total",
			Help: "Total operations by kind and terminal status",
		}, []string{"kind", "status"}),
		opsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "dback_operations_active",
			Help: "Currently running operations",
		}),
		tasksSkipped: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dback_tasks_skipped_total",
			Help: "Tasks skipped due to overlap or validation",
		}),
		notifyFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dback_notify_delivery_failures_total",
			Help: "Notification delivery failures (logged server-side)",
		}),
	}
	reg.MustRegister(c.opsTotal, c.opsActive, c.tasksSkipped, c.notifyFailed)
	return c
}

func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.reg, promhttp.HandlerOpts{})
}

func (c *Collector) Start(bus event.Bus) {
	subscribe := func(t event.Type, fn func(context.Context, event.Event)) {
		c.unsubs = append(c.unsubs, bus.Subscribe(t, func(ctx context.Context, ev event.Event) error {
			fn(ctx, ev)
			return nil
		}))
	}
	subscribe(event.TypeOperationStarted, func(_ context.Context, ev event.Event) {
		c.opsActive.Inc()
	})
	subscribe(event.TypeOperationCompleted, func(_ context.Context, ev event.Event) {
		c.opsActive.Dec()
		e := ev.(event.OperationCompleted)
		c.opsTotal.WithLabelValues(string(e.Kind), string(operation.StatusSucceeded)).Inc()
	})
	subscribe(event.TypeOperationFailed, func(_ context.Context, ev event.Event) {
		c.opsActive.Dec()
		e := ev.(event.OperationFailed)
		c.opsTotal.WithLabelValues(string(e.Kind), string(operation.StatusFailed)).Inc()
	})
	subscribe(event.TypeOperationCanceled, func(_ context.Context, ev event.Event) {
		c.opsActive.Dec()
		e := ev.(event.OperationCanceled)
		c.opsTotal.WithLabelValues(string(e.Kind), string(operation.StatusCanceled)).Inc()
	})
	subscribe(event.TypeTaskSkipped, func(_ context.Context, _ event.Event) {
		c.tasksSkipped.Inc()
	})
}

func (c *Collector) Stop() {
	for _, u := range c.unsubs {
		u()
	}
	c.unsubs = nil
}

func (c *Collector) IncNotifyFailure() {
	c.notifyFailed.Inc()
}
