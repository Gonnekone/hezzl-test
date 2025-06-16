package producer

import (
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/nats-io/nats.go"
	"log/slog"
)

type ProducerInterface interface {
	Send(data []byte) error
	SendAsync(data []byte) error
}

type Producer struct {
	nc *nats.Conn
	js nats.JetStreamContext

	cfg config.Nats
	log *slog.Logger
}

func New(log *slog.Logger, cfg config.Nats) (*Producer, error) {
	const op = "producer.New"

	nc, err := nats.Connect(cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("%s: connect to NATS: %w", op, err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("%s: create JetStream context: %w", op, err)
	}

	return &Producer{
		nc:  nc,
		js:  js,
		cfg: cfg,
		log: log,
	}, nil
}

func (p *Producer) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

func (p *Producer) Send(data []byte) error {
	ack, err := p.js.Publish(p.cfg.Subject, data)
	if err != nil {
		p.log.Error("failed to publish message", "error", err, "subject", p.cfg.Subject)
		return fmt.Errorf("publish message to %s: %w", p.cfg.Subject, err)
	}

	p.log.Debug("message published",
		"subject", p.cfg.Subject,
		"stream", ack.Stream,
		"sequence", ack.Sequence)

	return nil
}

func (p *Producer) SendAsync(data []byte) error {
	_, err := p.js.PublishAsync(p.cfg.Subject, data)
	if err != nil {
		p.log.Error("failed to publish async message", "error", err, "subject", p.cfg.Subject)
		return fmt.Errorf("publish async message to %s: %w", p.cfg.Subject, err)
	}

	p.log.Debug("async message published", "subject", p.cfg.Subject)
	return nil
}
