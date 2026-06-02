// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/GriffinGuard/Griffino-Go/internal/transport"
)

const replyQueueName = "amq.rabbitmq.reply-to"

// rabbitMQTransport is the default Transport implementation. It speaks the
// platform's RabbitMQ wire protocol, mirroring the Python SDK byte-for-byte:
// the same routing keys, queue names, exchange usage, reply matching, and the
// two provider delivery paths (synchronous invoke and blueprint task-node).
type rabbitMQTransport struct {
	cfg    Config
	logger *log.Logger

	mu      sync.Mutex
	started bool
	conn    *amqp.Connection
	ch      *amqp.Channel

	// pending maps a correlation id to the channel awaiting its reply.
	pendingMu sync.Mutex
	pending   map[string]chan map[string]any

	// consumerCtx is canceled by consumerCancel to stop the long-running
	// consumer goroutines.
	consumerCtx    context.Context
	consumerCancel context.CancelFunc
}

var _ Transport = (*rabbitMQTransport)(nil)

// newRabbitMQTransport builds a transport from cfg without connecting; the
// connection is established lazily on Start.
func newRabbitMQTransport(cfg Config) *rabbitMQTransport {
	return &rabbitMQTransport{
		cfg:     cfg.withDefaults(),
		logger:  log.New(log.Writer(), "griffino/transport ", log.LstdFlags),
		pending: make(map[string]chan map[string]any),
	}
}

// --- pure helpers (broker-free, unit-tested) ---

// invokeRoutingKey builds the routing key for a synchronous capability invoke.
func invokeRoutingKey(userID, capabilityType string) string {
	return fmt.Sprintf("invoke.griffino.router.%s.%s.v1", userID, capabilityType)
}

// dispatchRoutingKey builds the routing key for a fire-and-forget event.
func dispatchRoutingKey(userID, event string) string {
	return fmt.Sprintf("dispatch.griffino.router.%s.%s.v1", userID, event)
}

// nodeResultRoutingKey builds the routing key for a blueprint node result.
func nodeResultRoutingKey(userID string) string {
	return fmt.Sprintf("dispatch.griffino.router.%s.node.result.v1", userID)
}

// providerQueueName builds the queue name a provider consumes from.
func providerQueueName(pluginID, capabilityID string) string {
	return fmt.Sprintf("plugin.%s.%s", pluginID, capabilityID)
}

// consumerQueueName builds the durable queue name an event consumer binds to.
// Dots in the event become underscores, matching the platform.
func consumerQueueName(pluginID, event string) string {
	return fmt.Sprintf("plugin.%s.consumer.%s", pluginID, strings.ReplaceAll(event, ".", "_"))
}

// actionQueueName builds the queue name action deliveries arrive on.
func actionQueueName(pluginID string) string {
	return fmt.Sprintf("action.%s", pluginID)
}

// actionIDFromRoutingKey extracts the action id from an action routing key of
// the form action.{pluginId}.{userId}.{actionId}.v1. The plugin id may itself
// contain dots, so the action id is the segment immediately before the trailing
// ".v1". It returns "" when the routing key has too few segments.
func actionIDFromRoutingKey(routingKey string) string {
	if routingKey == "" {
		return ""
	}
	parts := strings.Split(routingKey, ".")
	if len(parts) < 5 {
		return ""
	}
	return parts[len(parts)-2]
}

// --- Transport implementation ---

// Start dials the broker, opens a channel, sets QoS, declares the exchanges
// passively, and begins consuming replies on the direct-reply-to queue. It is
// idempotent.
func (t *rabbitMQTransport) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started {
		return nil
	}

	url := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		t.cfg.RabbitMQUser, t.cfg.RabbitMQPassword, t.cfg.RabbitMQHost, t.cfg.RabbitMQPort)
	conn, err := amqp.Dial(url)
	if err != nil {
		return transportError(err, "failed to dial RabbitMQ broker")
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return transportError(err, "failed to open RabbitMQ channel")
	}
	if err := ch.Qos(16, 0, false); err != nil {
		_ = conn.Close()
		return transportError(err, "failed to set channel QoS")
	}
	if err := ch.ExchangeDeclarePassive(t.cfg.RouterExchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return transportError(err, "failed to declare router exchange %q", t.cfg.RouterExchange)
	}
	if err := ch.ExchangeDeclarePassive(t.cfg.PluginExchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return transportError(err, "failed to declare plugin exchange %q", t.cfg.PluginExchange)
	}

	// Direct reply-to: consume with autoAck=true. No explicit queue declaration
	// is needed for the pseudo-queue.
	replies, err := ch.Consume(replyQueueName, "", true, false, false, false, nil)
	if err != nil {
		_ = conn.Close()
		return transportError(err, "failed to consume reply queue")
	}

	consumerCtx, cancel := context.WithCancel(context.Background())
	t.conn = conn
	t.ch = ch
	t.consumerCtx = consumerCtx
	t.consumerCancel = cancel
	t.started = true

	go t.consumeReplies(consumerCtx, replies)
	return nil
}

// consumeReplies routes incoming reply deliveries to their waiting channels by
// correlation id.
func (t *rabbitMQTransport) consumeReplies(ctx context.Context, replies <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-replies:
			if !ok {
				return
			}
			decoded, err := transport.DecodeJSON(d.Body)
			if err != nil {
				t.logger.Printf("dropped reply: failed to decode body: %v", err)
				continue
			}
			if d.CorrelationId == "" {
				t.logger.Printf("dropped reply: missing correlation id")
				continue
			}
			t.pendingMu.Lock()
			waiter := t.pending[d.CorrelationId]
			t.pendingMu.Unlock()
			if waiter == nil {
				t.logger.Printf("dropped reply for unknown correlation id: %s", d.CorrelationId)
				continue
			}
			select {
			case waiter <- decoded:
			default:
			}
		}
	}
}

// Invoke publishes a synchronous capability request and waits for the reply,
// the deadline, or context cancellation.
func (t *rabbitMQTransport) Invoke(ctx context.Context, req InvokeRequest) (map[string]any, error) {
	if err := t.Start(ctx); err != nil {
		return nil, err
	}
	t.mu.Lock()
	ch := t.ch
	t.mu.Unlock()
	if ch == nil {
		return nil, transportError(nil, "RabbitMQ transport is not initialized")
	}

	msgID := transport.NewMessageID()
	envelope := transport.BuildInvokeEnvelope(req.TaskID, msgID, req.UserID, req.PluginID, req.Payload, req.Slot)
	body, err := transport.EncodeJSON(envelope)
	if err != nil {
		return nil, transportError(err, "failed to encode invoke envelope")
	}

	waiter := make(chan map[string]any, 1)
	t.pendingMu.Lock()
	t.pending[msgID] = waiter
	t.pendingMu.Unlock()
	defer func() {
		t.pendingMu.Lock()
		delete(t.pending, msgID)
		t.pendingMu.Unlock()
	}()

	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: msgID,
		MessageId:     msgID,
		ReplyTo:       replyQueueName,
		Expiration:    strconv.Itoa(req.TimeoutMS),
		DeliveryMode:  amqp.Persistent,
		Body:          body,
	}
	if err := ch.PublishWithContext(ctx, t.cfg.RouterExchange, invokeRoutingKey(req.UserID, req.CapabilityType), false, false, pub); err != nil {
		return nil, transportError(err, "failed to publish invoke request")
	}

	timer := time.NewTimer(time.Duration(req.TimeoutMS) * time.Millisecond)
	defer timer.Stop()
	select {
	case reply := <-waiter:
		return reply, nil
	case <-timer.C:
		return nil, timeoutError(nil, "provider_timeout")
	case <-ctx.Done():
		return nil, transportError(ctx.Err(), "invoke canceled")
	}
}

// Dispatch publishes a fire-and-forget event to the router exchange.
func (t *rabbitMQTransport) Dispatch(ctx context.Context, req DispatchRequest) error {
	if err := t.Start(ctx); err != nil {
		return err
	}
	t.mu.Lock()
	ch := t.ch
	t.mu.Unlock()
	if ch == nil {
		return transportError(nil, "RabbitMQ transport is not initialized")
	}

	msgID := transport.NewMessageID()
	envelope := transport.BuildDispatchEnvelope(req.TaskID, msgID, req.UserID, req.PluginID, req.Event, req.Payload)
	body, err := transport.EncodeJSON(envelope)
	if err != nil {
		return transportError(err, "failed to encode dispatch envelope")
	}
	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: msgID,
		MessageId:     msgID,
		DeliveryMode:  amqp.Persistent,
		Body:          body,
	}
	if err := ch.PublishWithContext(ctx, t.cfg.RouterExchange, dispatchRoutingKey(req.UserID, req.Event), false, false, pub); err != nil {
		return transportError(err, "failed to publish dispatch event")
	}
	return nil
}

// StartProviderConsumers consumes each provider's queue, handling both the
// synchronous-invoke and blueprint-task-node delivery paths.
func (t *rabbitMQTransport) StartProviderConsumers(ctx context.Context, regs []ProviderRegistration, run HandlerRunner) error {
	if err := t.Start(ctx); err != nil {
		return err
	}
	t.mu.Lock()
	ch := t.ch
	cctx := t.consumerContext()
	t.mu.Unlock()
	if ch == nil {
		return transportError(nil, "RabbitMQ transport is not initialized")
	}

	for _, reg := range regs {
		queue := providerQueueName(t.cfg.PluginID, reg.CapabilityID)
		if _, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil); err != nil {
			return transportError(err, "failed to declare provider queue %q", queue)
		}
		deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			return transportError(err, "failed to consume provider queue %q", queue)
		}
		go t.runProviderConsumer(cctx, reg, run, deliveries)
	}
	return nil
}

func (t *rabbitMQTransport) runProviderConsumer(ctx context.Context, reg ProviderRegistration, run HandlerRunner, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			t.handleProviderDelivery(ctx, reg, run, d)
		}
	}
}

func (t *rabbitMQTransport) handleProviderDelivery(ctx context.Context, reg ProviderRegistration, run HandlerRunner, d amqp.Delivery) {
	defer func() { _ = d.Ack(false) }()

	hc, err := t.decodeContext(d)
	if err != nil {
		t.logger.Printf("provider %s: %v", reg.CapabilityID, err)
		return
	}

	// Path B (blueprint task-node): no reply_to and a nodeId present. Signal
	// completion by dispatching a node-result envelope.
	if d.ReplyTo == "" && hc.NodeID != "" {
		result, herr := run(reg.Handler, hc)
		if herr != nil {
			t.logger.Printf("provider %s: handler failed for task-node %s: %v", reg.CapabilityID, hc.NodeID, herr)
			t.dispatchNodeResult(ctx, hc, false, map[string]any{}, herr.Error())
			return
		}
		output := result
		if output == nil {
			output = map[string]any{}
		}
		t.dispatchNodeResult(ctx, hc, true, output, "")
		return
	}

	// Path A (synchronous invoke): reply via reply_to/correlation_id.
	var response map[string]any
	result, herr := run(reg.Handler, hc)
	if herr != nil {
		t.logger.Printf("provider %s: handler failed: %v", reg.CapabilityID, herr)
		response = transport.BuildProviderError("provider_execution_failed")
	} else if result != nil {
		response = result
	} else {
		response = map[string]any{}
	}
	t.publishReply(ctx, d, response)
}

func (t *rabbitMQTransport) dispatchNodeResult(ctx context.Context, hc *HandlerContext, ok bool, output map[string]any, failReason string) {
	t.mu.Lock()
	ch := t.ch
	t.mu.Unlock()
	if ch == nil {
		t.logger.Printf("cannot dispatch node result: transport not initialized")
		return
	}
	msgID := transport.NewMessageID()
	envelope := transport.BuildNodeResultEnvelope(hc.TaskID, msgID, hc.UserID, t.cfg.PluginID, hc.NodeID, ok, output, failReason)
	body, err := transport.EncodeJSON(envelope)
	if err != nil {
		t.logger.Printf("failed to encode node result: %v", err)
		return
	}
	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: msgID,
		MessageId:     msgID,
		DeliveryMode:  amqp.Persistent,
		Body:          body,
	}
	if err := ch.PublishWithContext(ctx, t.cfg.RouterExchange, nodeResultRoutingKey(hc.UserID), false, false, pub); err != nil {
		t.logger.Printf("failed to publish node result: %v", err)
	}
}

func (t *rabbitMQTransport) publishReply(ctx context.Context, request amqp.Delivery, payload map[string]any) {
	if request.ReplyTo == "" {
		t.logger.Printf("provider request did not include reply_to queue")
		return
	}
	t.mu.Lock()
	ch := t.ch
	t.mu.Unlock()
	if ch == nil {
		t.logger.Printf("cannot publish reply: transport not initialized")
		return
	}
	body, err := transport.EncodeJSON(payload)
	if err != nil {
		t.logger.Printf("failed to encode reply: %v", err)
		return
	}
	pub := amqp.Publishing{
		ContentType:   "application/json",
		CorrelationId: request.CorrelationId,
		DeliveryMode:  amqp.Persistent,
		Body:          body,
	}
	// Reply on the default exchange, routed directly to the reply queue.
	if err := ch.PublishWithContext(ctx, "", request.ReplyTo, false, false, pub); err != nil {
		t.logger.Printf("failed to publish reply: %v", err)
	}
}

// StartEventConsumers declares and consumes a durable queue per event consumer,
// running handlers fire-and-forget.
func (t *rabbitMQTransport) StartEventConsumers(ctx context.Context, regs []ConsumerRegistration, run HandlerRunner) error {
	if err := t.Start(ctx); err != nil {
		return err
	}
	t.mu.Lock()
	ch := t.ch
	cctx := t.consumerContext()
	t.mu.Unlock()
	if ch == nil {
		return transportError(nil, "RabbitMQ transport is not initialized")
	}

	for _, reg := range regs {
		queue := consumerQueueName(t.cfg.PluginID, reg.Event)
		if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			return transportError(err, "failed to declare consumer queue %q", queue)
		}
		deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			return transportError(err, "failed to consume consumer queue %q", queue)
		}
		go t.runEventConsumer(cctx, reg, run, deliveries)
	}
	return nil
}

func (t *rabbitMQTransport) runEventConsumer(ctx context.Context, reg ConsumerRegistration, run HandlerRunner, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			t.handleEventDelivery(reg, run, d)
		}
	}
}

func (t *rabbitMQTransport) handleEventDelivery(reg ConsumerRegistration, run HandlerRunner, d amqp.Delivery) {
	defer func() { _ = d.Ack(false) }()
	hc, err := t.decodeContext(d)
	if err != nil {
		t.logger.Printf("consumer %s: %v", reg.Event, err)
		return
	}
	if _, herr := run(reg.Handler, hc); herr != nil {
		t.logger.Printf("consumer %s: handler failed: %v", reg.Event, herr)
	}
}

// StartActionConsumers declares and consumes the single per-plugin action queue,
// dispatching each delivery to the handler matching its routing-key action id.
func (t *rabbitMQTransport) StartActionConsumers(ctx context.Context, regs []ActionRegistration, run HandlerRunner) error {
	if len(regs) == 0 {
		return nil
	}
	if err := t.Start(ctx); err != nil {
		return err
	}
	t.mu.Lock()
	ch := t.ch
	cctx := t.consumerContext()
	t.mu.Unlock()
	if ch == nil {
		return transportError(nil, "RabbitMQ transport is not initialized")
	}

	queue := actionQueueName(t.cfg.PluginID)
	if _, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil); err != nil {
		return transportError(err, "failed to declare action queue %q", queue)
	}
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return transportError(err, "failed to consume action queue %q", queue)
	}

	byID := make(map[string]ActionRegistration, len(regs))
	for _, reg := range regs {
		byID[reg.ActionID] = reg
	}
	go t.runActionConsumer(cctx, byID, run, deliveries)
	return nil
}

func (t *rabbitMQTransport) runActionConsumer(ctx context.Context, byID map[string]ActionRegistration, run HandlerRunner, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			t.handleActionDelivery(byID, run, d)
		}
	}
}

func (t *rabbitMQTransport) handleActionDelivery(byID map[string]ActionRegistration, run HandlerRunner, d amqp.Delivery) {
	defer func() { _ = d.Ack(false) }()
	actionID := actionIDFromRoutingKey(d.RoutingKey)
	reg, ok := byID[actionID]
	if !ok {
		t.logger.Printf("dropped action message with unknown action id %q (routing key %q)", actionID, d.RoutingKey)
		return
	}
	hc, err := t.decodeContext(d)
	if err != nil {
		t.logger.Printf("action %s: %v", actionID, err)
		return
	}
	hc.ActionID = actionID
	if _, herr := run(reg.Handler, hc); herr != nil {
		t.logger.Printf("action %s: handler failed: %v", actionID, herr)
	}
}

// decodeContext decodes a delivery body into a HandlerContext.
func (t *rabbitMQTransport) decodeContext(d amqp.Delivery) (*HandlerContext, error) {
	decoded, err := transport.DecodeJSON(d.Body)
	if err != nil {
		return nil, transportError(err, "failed to decode incoming message")
	}
	hc, err := contextFromMessage(decoded, t.cfg.PluginID)
	if err != nil {
		return nil, transportError(err, "failed to build handler context")
	}
	return hc, nil
}

// consumerContext returns the context tied to the transport's consumer
// lifetime; it is canceled by Close. The caller must hold t.mu.
func (t *rabbitMQTransport) consumerContext() context.Context {
	if t.consumerCtx == nil {
		return context.Background()
	}
	return t.consumerCtx
}

// Close cancels consumers, closes the channel and connection, and fails any
// pending replies.
func (t *rabbitMQTransport) Close(ctx context.Context) error {
	t.mu.Lock()
	if !t.started {
		t.mu.Unlock()
		return nil
	}
	cancel := t.consumerCancel
	ch := t.ch
	conn := t.conn
	t.ch = nil
	t.conn = nil
	t.consumerCtx = nil
	t.consumerCancel = nil
	t.started = false
	t.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	var firstErr error
	if ch != nil {
		if err := ch.Close(); err != nil {
			firstErr = transportError(err, "failed to close channel")
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = transportError(err, "failed to close connection")
		}
	}

	t.pendingMu.Lock()
	for id, waiter := range t.pending {
		close(waiter)
		delete(t.pending, id)
	}
	t.pendingMu.Unlock()

	return firstErr
}
