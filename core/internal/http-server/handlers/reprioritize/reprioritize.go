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

const errCode = 3

//go:generate go run github.com/vektra/mockery/v2@v2.42.1 --name=URLSaver
type GoodPriorityUpdater interface {
	UpdateGoodsPriority(
		ctx context.Context,
		id string,
		projectID string,
		priority int,
	) ([]models.Good, error)
	InvalidList(ctx context.Context) error
}

type GoodPriorityView struct {
	ID       int `json:"id"`
	Priority int `json:"priority"`
}

type Request struct {
	NewPriority int `json:"newPriority" validate:"required"`
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

		log = log.With(
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

		log.Info("request body decoded", slog.Any("request_body", req))

		id := r.URL.Query().Get("id")
		if id == "" {
			log.Info("id is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		projectID := r.URL.Query().Get("projectId")
		if projectID == "" {
			log.Info("projectId is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		if err := validator.New().Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			errors.As(err, &validateErr)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		goods, err := goodPriorityUpdater.UpdateGoodsPriority(
			r.Context(),
			id,
			projectID,
			req.NewPriority,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error("failed to delete good", sl.Err(err))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, ErrorResponse{
				Code:    errCode,
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

		goodsPriority := make([]GoodPriorityView, 0, len(goods))

		for _, good := range goods {
			data, err := json.Marshal(good)
			if err != nil {
				log.Warn("failed to marshal data", sl.Err(err))
			}

			// почему не сделать просто eventlogs?
			err = producer.SendAsync(data)
			if err != nil {
				log.Warn("failed to send message to nats", sl.Err(err))
			}

			goodsPriority = append(goodsPriority, GoodPriorityView{
				ID:       good.ID,
				Priority: good.Priority,
			})
		}

		render.JSON(w, r, Response{Priorities: goodsPriority})
	}
}
