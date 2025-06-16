package list

import (
	"context"
	resp "github.com/Gonnekone/hezzl-test/core/internal/lib/api/response"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/sl"
	"github.com/Gonnekone/hezzl-test/core/internal/models"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"strconv"
)

const limitDefault = "10"
const offsetDefault = "1"

//go:generate go run github.com/vektra/mockery/v3@v3.4.0 --name=GoodLister
type GoodLister interface {
	ListGoods(
		ctx context.Context,
		limit int,
		offset int,
	) (*GoodListResponse, error)
	GetCachedList(ctx context.Context, limit, offset int) ([]byte, error)
	SaveListInCache(ctx context.Context, list GoodListResponse) error
}

type GoodListResponse struct {
	Meta  GoodMetaListResponse `json:"meta"`
	Goods []models.Good        `json:"goods"`
}

type GoodMetaListResponse struct {
	Total   int `json:"total"`
	Removed int `json:"removed"`
	Limit   int `json:"limit"`
	Offset  int `json:"offset"`
}

func New(log *slog.Logger, goodLister GoodLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.list.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		limit, offset, err := retrieveLimitAndOffset(r)
		if err != nil {
			log.Error("failed to retrieve limit and offset", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to retrieve limit and offset"))

			return
		}

		if data, err := goodLister.GetCachedList(r.Context(), limit, offset); err == nil {
			log.Info("goods listed from cache successfully")

			w.Header().Set("Content-Type", "application/json")

			//nolint: errcheck
			w.Write(data)

			return
		}

		goods, err := goodLister.ListGoods(r.Context(), limit, offset)
		if err != nil {
			log.Error("failed to list goods", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to list goods"))

			return
		}

		err = goodLister.SaveListInCache(r.Context(), *goods)
		if err != nil {
			log.Warn("failed to cache list", sl.Err(err))
		}

		log.Info("goods listed from main storage successfully")

		render.JSON(w, r, goods)
	}
}

func retrieveLimitAndOffset(r *http.Request) (int, int, error) {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = limitDefault
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offsetStr = offsetDefault
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return 0, 0, err
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return 0, 0, err
	}

	return limit, offset, nil
}
