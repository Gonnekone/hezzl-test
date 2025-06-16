package remove

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
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
)

const errCode = 3

//go:generate go run github.com/vektra/mockery/v2@v2.42.1 --name=URLDeleter
type GoodDeleter interface {
	DeleteGood(
		ctx context.Context,
		id string,
		projectID string,
	) (*models.Good, error)
	InvalidList(ctx context.Context) error
}

type Response struct {
	ID        int  `json:"id"`
	ProjectID int  `json:"projectId"`
	Removed   bool `json:"removed"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"message"`
	Details string `json:"details"`
}

func New(
	log *slog.Logger,
	goodDeleter GoodDeleter,
	producer producer.ProducerInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.remove.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := r.URL.Query().Get("id")
		if id == "" {
			log.Info("id is empty")

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		projectID := r.URL.Query().Get("projectId")
		if projectID == "" {
			log.Info("projectId is empty")

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("invalid url params"))

			return
		}

		good, err := goodDeleter.DeleteGood(r.Context(), id, projectID)
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
			log.Error("failed to delete good", sl.Err(err))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("failed to delete good"))

			return
		}

		err = goodDeleter.InvalidList(r.Context())
		if err != nil {
			log.Error("failed to invalid cached list", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)

			return
		}

		log.Info("good deleted successfully")

		data, err := json.Marshal(good)
		if err != nil {
			log.Warn("failed to marshal data", sl.Err(err))
		}

		err = producer.SendAsync(data)
		if err != nil {
			log.Warn("failed to send message to nats", sl.Err(err))
		}

		render.JSON(w, r, Response{
			ID:        good.ID,
			ProjectID: good.ProjectID,
			Removed:   good.Removed,
		})
	}
}
