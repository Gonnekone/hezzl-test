package producer

import (
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/config"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/nats-io/nats.go"
	"log/slog"
)

//go:generate mockgen -source=producer.go -destination=mocks/producer.go -package=mocks
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
		p.log.Error("failed to publish message",
			sl.Err(err),
			slog.String("subject", p.cfg.Subject),
		)
		return fmt.Errorf("publish message to %s: %w", p.cfg.Subject, err)
	}

	p.log.Debug("message published",
		slog.String("subject", p.cfg.Subject),
		slog.String("string", ack.Stream),
		slog.Uint64("seq", ack.Sequence),
	)

	return nil
}

func (p *Producer) SendAsync(data []byte) error {
	_, err := p.js.PublishAsync(p.cfg.Subject, data)
	if err != nil {
		p.log.Error("failed to publish async message",
			sl.Err(err),
			slog.String("subject", p.cfg.Subject),
		)
		return fmt.Errorf("publish async message to %s: %w", p.cfg.Subject, err)
	}

	p.log.Debug("async message published", slog.String("subject", p.cfg.Subject))
	return nil
}
