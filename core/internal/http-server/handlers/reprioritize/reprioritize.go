package reprioritize

import (
	"context"
	"encoding/json"
	"errors"
	resp "github.com/Gonnekone/hezzl-test/core/internal/lib/api/response"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/Gonnekone/hezzl-test/core/internal/models"
	"github.com/Gonnekone/hezzl-test/core/internal/producer"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
)

//go:generate go run github.com/vektra/mockery/v2@v2.42.1 --name=URLSaver
type GoodPriorityUpdater interface {
	UpdateGoodsPriority(ctx context.Context, id string, projectId string, priority int) ([]GoodPriorityView, error)
	InvalidList(ctx context.Context) error
	GetGood(ctx context.Context, id string, projectId string) (*models.Good, error)
}

type GoodPriorityView struct {
	Id       int `json:"id"`
	Priority int `json:"priority"`
}

type Request struct {
	NewPriority int `json:"newPriority"`
}

type Response struct {
	Priorities []GoodPriorityView `json:"priorities"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"message"`
	Details string `json:"details"`
}

func New(
	log *slog.Logger,
	goodPriorityUpdater GoodPriorityUpdater,
	producer producer.ProducerInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reprioritize.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to decode request body"))

			return
		}

		log.Info("request body decoded", slog.Any("request body", req))

		id := r.URL.Query().Get("id")
		if id == "" {
			log.Info("id is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		projectId := r.URL.Query().Get("projectId")
		if projectId == "" {
			log.Info("projectId is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		goodsPriority, err := goodPriorityUpdater.UpdateGoodsPriority(r.Context(), id, projectId, req.NewPriority)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error("failed to delete good", sl.Err(err))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrorResponse{
				Code:    3,
				Msg:     "errors.common.notFound",
				Details: err.Error(),
			})

			return
		}

		if err != nil {
			log.Error("failed to update priority", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to update priority"))

			return
		}

		err = goodPriorityUpdater.InvalidList(r.Context())
		if err != nil {
			log.Error("failed to invalid cached list", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)

			return
		}

		log.Info("priority updated successfully")

		// Можно возвращать все измененные товары, но тогда это уже не логи будут,
		// а копия хранилища, не вижу в этом смысла. Да там будут неверные приоритеты,
		// но зачем там вообще все данные о товарах, почему не создать евент логи
		// что товар был создан - изменен - удален... Без конкретики.
		good, err := goodPriorityUpdater.GetGood(r.Context(), id, projectId)
		if err != nil {
			log.Warn("failed to get good", sl.Err(err))
		}

		data, err := json.Marshal(good)
		if err != nil {
			log.Warn("failed to marshal data", sl.Err(err))
		}

		err = producer.SendAsync(data)
		if err != nil {
			log.Warn("failed to send message to nats", sl.Err(err))
		}

		// здесь мы еще раз маршаллим, по хорошему просто передавать []byte
		render.JSON(w, r, Response{Priorities: goodsPriority})
	}
}
