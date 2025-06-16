package listener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/config"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/lib/logger/sl"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/models"
	"github.com/Gonnekone/hezzl-test/clickhouse-service/internal/storage/clickhouse"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type Listener struct {
	nc *nats.Conn
	js nats.JetStreamContext

	clh *clickhouse.ClickHouseStorage

	log *slog.Logger
	cfg config.Nats

	goodsBatch []models.Good
	batchCh    chan models.Good
	batchTimer time.Duration
}

func New(log *slog.Logger, cfg config.Nats, clh *clickhouse.ClickHouseStorage) (*Listener, error) {
	nc, err := nats.Connect(cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return &Listener{
		js:         js,
		nc:         nc,
		clh:        clh,
		cfg:        cfg,
		log:        log,
		batchTimer: clh.BatchTimer,
		goodsBatch: make([]models.Good, 0, cfg.BatchSize),
		batchCh:    make(chan models.Good, cfg.BatchSize*2), //nolint: mnd
	}, nil
}

func (l *Listener) Close() {
	if l.nc != nil {
		l.nc.Close()
	}
	close(l.batchCh)
}

func (l *Listener) Start(ctx context.Context) error {
	_, err := l.js.AddConsumer(l.cfg.StreamName, &nats.ConsumerConfig{
		Durable:       l.cfg.ConsumerName,
		DeliverPolicy: nats.DeliverNewPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    1,
		AckWait:       l.cfg.AckWait,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	sub, err := l.js.PullSubscribe(l.cfg.Subject, l.cfg.ConsumerName)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	l.log.Info("Started listening for messages",
		slog.String("stream", l.cfg.StreamName),
		slog.String("subject", l.cfg.Subject),
		slog.Int("batch_size", l.cfg.BatchSize))

	go l.batchProcessor(ctx)

	go l.listen(ctx, sub)

	return nil
}

func (l *Listener) listen(ctx context.Context, sub *nats.Subscription) {
	defer sub.Unsubscribe() //nolint: errcheck

	for {
		select {
		case <-ctx.Done():
			l.log.Debug("Shutting down listener...")
			l.flushBatch(ctx)
			return
		default:
			msgs, err := sub.Fetch(l.cfg.BatchSize, nats.MaxWait(2*time.Second)) //nolint: mnd
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) {
					continue
				}
				l.log.Warn("Error fetching messages", sl.Err(err))
				time.Sleep(1 * time.Second)
				continue
			}

			processedMsgs := make([]*nats.Msg, 0, len(msgs))
			for _, msg := range msgs {
				good, err := l.processMessage(msg)
				if err != nil {
					l.log.Warn("Error processing message", sl.Err(err))
					continue
				}

				select {
				case l.batchCh <- *good:
					processedMsgs = append(processedMsgs, msg)
				case <-ctx.Done():
					return
				default:
					l.log.Warn("Batch channel full, dropping message")
				}
			}

			// ACK'аем только успешно обработанные сообщения
			for _, msg := range processedMsgs {
				if err := msg.Ack(); err != nil {
					l.log.Warn("Error acknowledging message", sl.Err(err))
				}
			}
		}
	}
}

func (l *Listener) processMessage(msg *nats.Msg) (*models.Good, error) {
	l.log.Debug("Processing message", slog.String("data", string(msg.Data)))

	var good models.Good
	if err := json.Unmarshal(msg.Data, &good); err != nil {
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return &good, nil
}

// Batch processor - обрабатывает накопленные сообщения
func (l *Listener) batchProcessor(ctx context.Context) {
	timer := time.NewTimer(l.batchTimer)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			l.flushBatch(ctx)
			return

		case good, ok := <-l.batchCh:
			if !ok {
				return
			}

			l.goodsBatch = append(l.goodsBatch, good)

			if len(l.goodsBatch) >= l.cfg.BatchSize {
				l.flushBatch(ctx)
				timer.Reset(l.batchTimer)
			}

		case <-timer.C:
			if len(l.goodsBatch) > 0 {
				l.flushBatch(ctx)
			}
			timer.Reset(l.batchTimer)
		}
	}
}

func (l *Listener) flushBatch(ctx context.Context) {
	if len(l.goodsBatch) == 0 {
		return
	}

	start := time.Now()
	err := l.clh.LogGoods(ctx, l.goodsBatch)
	duration := time.Since(start)

	if err != nil {
		l.log.Error("Failed to flush batch to ClickHouse",
			sl.Err(err),
			slog.Int("batch_size", len(l.goodsBatch)),
			slog.Duration("duration", duration))
	} else {
		l.log.Info("Successfully flushed batch to ClickHouse",
			slog.Int("batch_size", len(l.goodsBatch)),
			slog.Duration("duration", duration))
	}

	l.goodsBatch = l.goodsBatch[:0]
}
