package update

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
type GoodUpdater interface {
	UpdateGood(
		ctx context.Context,
		id string,
		projectID string,
		name string,
		desc string,
	) (*models.Good, error)
	InvalidList(ctx context.Context) error
}

type Request struct {
	Name string `json:"name" validate:"required"`
	Desc string `json:"description,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"message"`
	Details string `json:"details"`
}

func New(
	log *slog.Logger,
	goodUpdater GoodUpdater,
	producer producer.ProducerInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.update.New"

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

		good, err := goodUpdater.UpdateGood(r.Context(), id, projectID, req.Name, req.Desc)
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
			log.Error("failed to update good", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to update good"))

			return
		}

		err = goodUpdater.InvalidList(r.Context())
		if err != nil {
			log.Error("failed to invalid cached list", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)

			return
		}

		log.Info("good added successfully", slog.Any("good", good))

		data, err := json.Marshal(good)
		if err != nil {
			log.Warn("failed to marshal data", sl.Err(err))
		}

		err = producer.SendAsync(data)
		if err != nil {
			log.Warn("failed to send message to nats", sl.Err(err))
		}

		// здесь мы еще раз маршаллим, по хорошему просто передавать []byte
		render.JSON(w, r, good)
	}
}
