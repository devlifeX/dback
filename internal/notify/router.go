package notify

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"dback/internal/event"
	"dback/internal/metrics"
	"dback/models"
)

type ChannelStore interface {
	ListNotifyChannels() ([]models.NotifyChannel, error)
	GetNotifyChannel(id string) (models.NotifyChannel, error)
}

type Router struct {
	bus      event.Bus
	store    ChannelStore
	namer    HostNamer
	registry *Registry
	metrics  *metrics.Collector
	timeout  time.Duration
	unsubs   []func()
}

func NewRouter(bus event.Bus, store ChannelStore, registry *Registry, namer HostNamer) *Router {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Router{
		bus:      bus,
		store:    store,
		namer:    namer,
		registry: registry,
		timeout:  30 * time.Second,
	}
}

func (r *Router) SetMetrics(c *metrics.Collector) {
	r.metrics = c
}

func (r *Router) Start() {
	types := []event.Type{
		event.TypeOperationStarted,
		event.TypeOperationCompleted,
		event.TypeOperationFailed,
		event.TypeOperationCanceled,
		event.TypeTaskSkipped,
	}
	for _, t := range types {
		typ := t
		unsub := r.bus.Subscribe(typ, func(ctx context.Context, ev event.Event) error {
			r.handle(ctx, ev)
			return nil
		})
		r.unsubs = append(r.unsubs, unsub)
	}
}

func (r *Router) Stop() {
	for _, unsub := range r.unsubs {
		unsub()
	}
	r.unsubs = nil
}

func (r *Router) Test(ctx context.Context, channelID string) error {
	ch, err := r.store.GetNotifyChannel(channelID)
	if err != nil {
		return err
	}
	if err := ValidateChannel(ch); err != nil {
		return err
	}
	sender, ok := r.registry.Sender(ch.Provider)
	if !ok {
		return fmt.Errorf("no sender for provider %q", ch.Provider)
	}
	return SendWithRetry(ctx, func(callCtx context.Context) error {
		return sender.Send(callCtx, ch.Config, TestMessage())
	}, DefaultRetryDelays())
}

func (r *Router) handle(_ context.Context, ev event.Event) {
	msg, ok := MessageFromEvent(ev, r.namer)
	if !ok {
		return
	}
	channels, err := r.store.ListNotifyChannels()
	if err != nil {
		log.Printf("notify: list channels: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, ch := range channels {
		if !ch.Enabled || !ch.SubscribesTo(string(msg.Event)) {
			continue
		}
		channel := ch
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.deliver(context.Background(), channel, msg)
		}()
	}
	wg.Wait()
}

func (r *Router) deliver(parent context.Context, ch models.NotifyChannel, msg Message) {
	sender, ok := r.registry.Sender(ch.Provider)
	if !ok {
		log.Printf("notify: unknown provider %q for channel %s", ch.Provider, ch.ID)
		return
	}
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	err := SendWithRetry(ctx, func(callCtx context.Context) error {
		return sender.Send(callCtx, ch.Config, msg)
	}, DefaultRetryDelays())
	if err != nil {
		log.Printf("notify: channel %s (%s) delivery failed: %v", ch.Name, ch.ID, err)
		if r.metrics != nil {
			r.metrics.IncNotifyFailure()
		}
	}
}
