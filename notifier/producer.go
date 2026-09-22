// Package notifier turns typed business inputs into email delivery
// requests, routed by notification topic.
package notifier

import (
	"context"
	"fmt"

	"github.com/gloveboxhq/glovebox-go-code-challenge/notifier/channels/email"
)

type Producer struct {
	email         email.MailProvider
	topicBuilders map[string]TopicRequestBuilder
}

// ProducerProvider is the producer behavior callers can depend on for injection.
// Implementations support both topic construction and prebuilt requests.
type ProducerProvider interface {
	NotifyTopic(ctx context.Context, topic string, input any) error
	Notify(ctx context.Context, req Request) error
}

// TopicRequestBuilder builds delivery requests from a topic's business input.
// Built-in registration is currently owned by NewProducer; this interface does
// not imply a public runtime registration API.
type TopicRequestBuilder interface {
	Topic() string
	BuildRequest(ctx context.Context, input any) (Request, error)
}

// Request contains a prebuilt delivery request. Producers do not deep-copy Vars.
// Callers retain ownership of nested mutable values and must synchronize access.
type Request struct {
	Topic      string
	Recipients []string
	Template   email.TplID
	Vars       map[string]any
}

// defaultTopicBuilders lists every builder the producer knows about. Adding a
// new topic only requires appending its builder here.
func defaultTopicBuilders() []TopicRequestBuilder {
	return []TopicRequestBuilder{
		docUploadTopicBuilder,
		otpLoginTopicBuilder,
		policyRenewalTopicBuilder,
	}
}

// NewProducer returns a Producer that delivers through emailProvider and
// knows every topic listed in defaultTopicBuilders.
func NewProducer(emailProvider email.MailProvider) *Producer {
	builders := defaultTopicBuilders()
	p := &Producer{
		email:         emailProvider,
		topicBuilders: make(map[string]TopicRequestBuilder, len(builders)),
	}
	for _, b := range builders {
		p.topicBuilders[b.Topic()] = b
	}
	return p
}

// NotifyTopic builds the Request for topic from input and delivers it.
// It returns ErrTopicNotRegistered (wrapped) for an unknown topic, the
// builder's validation error for a bad input, or the delivery error.
func (p *Producer) NotifyTopic(ctx context.Context, topic string, input any) error {
	builder, ok := p.topicBuilders[topic]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTopicNotRegistered, topic)
	}
	req, err := builder.BuildRequest(ctx, input)
	if err != nil {
		return err
	}
	return p.Notify(ctx, req)
}

// Notify delivers a validated request through the configured provider.
func (p *Producer) Notify(ctx context.Context, req Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := email.RequireRecipient(req.Recipients); err != nil {
		return fmt.Errorf("notify %s: %w: %w", req.Topic, ErrMissingRecipients, err)
	}
	if req.Template == "" {
		return fmt.Errorf("%w: %s", ErrMissingTemplate, req.Topic)
	}
	return p.email.Send(req.Recipients, req.Template, req.Vars)
}

// topicBuilder is the shared skeleton of every topic: it checks the dynamic
// input type, delegates validation and template variables to build, and
// assembles the Request. Each topic file only supplies the static fields and
// its build function.
type topicBuilder[T any] struct {
	topic           string
	tpl             email.TplID
	errInvalidInput error
	// build validates in and returns the recipient plus template variables.
	build func(in T) (recipient string, vars map[string]any, err error)
}

// Topic returns the notification topic key this builder handles.
func (b topicBuilder[T]) Topic() string { return b.topic }

// BuildRequest validates input and builds the email Request for the topic.
func (b topicBuilder[T]) BuildRequest(_ context.Context, input any) (Request, error) {
	typed, ok := input.(T)
	if !ok {
		return Request{}, fmt.Errorf("%w: expected %T, got %T", b.errInvalidInput, typed, input)
	}
	recipient, vars, err := b.build(typed)
	if err != nil {
		return Request{}, err
	}
	return Request{
		Topic:      b.topic,
		Recipients: []string{recipient},
		Template:   b.tpl,
		Vars:       vars,
	}, nil
}
