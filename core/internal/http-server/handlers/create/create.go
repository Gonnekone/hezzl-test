package create

import (
	"context"
	"encoding/json"
	"fmt"
	resp "github.com/Gonnekone/hezzl-test/core/internal/lib/api/response"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/Gonnekone/hezzl-test/core/internal/models"
	"github.com/Gonnekone/hezzl-test/core/internal/producer"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
)

//go:generate go run github.com/vektra/mockery/v2@v2.42.1 --name=URLSaver
type GoodSaver interface {
	SaveGood(ctx context.Context, name string, projectId string) (*models.Good, error)
	InvalidList(ctx context.Context) error
}

type Request struct {
	Name string `json:"name" validate:"required"`
}

func New(
	log *slog.Logger,
	goodSaver GoodSaver,
	producer producer.ProducerInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.create.New"

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

		projectId := r.URL.Query().Get("projectId")
		if projectId == "" {
			log.Info("projectId is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		log.Info("request body decoded", slog.Any("request body", req))

		good, err := goodSaver.SaveGood(r.Context(), req.Name, projectId)
		if err != nil {
			log.Error("failed to save good", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to save good"))

			return
		}

		err = goodSaver.InvalidList(r.Context())
		if err != nil {
			log.Error("failed to invalid cached list", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)

			return
		}

		log.Info("good added", slog.Any("good", good))

		data, err := json.Marshal(good)
		if err != nil {
			log.Warn("failed to marshal data", sl.Err(err))
		}

		fmt.Println("producer: ", producer)
		log.Debug("producer: ", producer)
		err = producer.SendAsync(data)
		if err != nil {
			log.Warn("failed to send message to nats", sl.Err(err))
		}

		// здесь мы еще раз маршаллим, по хорошему просто передавать []byte
		render.JSON(w, r, good)
	}
}
