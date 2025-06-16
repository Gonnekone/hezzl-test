package create_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/create"
	"github.com/Gonnekone/hezzl-test/core/internal/http-server/handlers/create/mocks"
	"github.com/Gonnekone/hezzl-test/core/internal/lib/logger/handlers/slogdiscard"
	"github.com/Gonnekone/hezzl-test/core/internal/models"
	producer_mocks "github.com/Gonnekone/hezzl-test/core/internal/producer/mocks"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSaveHandler(t *testing.T) {
	storageErr := errors.New("storage error")

	type goodSaverMock struct {
		name      string
		projectID string

		resp *models.Good
		err  error
	}

	type invalidCacheMock struct {
		err error
	}

	type producerMock struct {
		err error
	}

	cases := []struct {
		name             string
		goodSaverMock    *goodSaverMock
		invalidCacheMock *invalidCacheMock
		producerMock     *producerMock
		reqBody          string
		projectID        string

		wantStatus int
		wantBody   string
	}{
		{
			name: "Success",
			goodSaverMock: &goodSaverMock{
				name:      "Apple",
				projectID: "1",
				resp: &models.Good{
					ID:          1,
					ProjectID:   1,
					Name:        "Apple",
					Description: "NO DESC",
					Priority:    5,
					Removed:     false,
					CreatedAt:   time.UnixMilli(1234567890),
				},
			},
			invalidCacheMock: &invalidCacheMock{},
			producerMock:     &producerMock{},
			reqBody:          `{"name":"Apple"}`,
			projectID:        "1",
			wantBody: `{
"id":1,"projectId":1,"name":"Apple",
"description":"NO DESC","priority":5,
"removed":false,"createdAt":"1970-01-15T09:56:07.89+03:00"
}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Empty name",
			reqBody:    `{"name":""}`,
			projectID:  "1",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"field Name is a required field", "status":"Error"}`,
		},
		{
			name: "SaveGood error",
			goodSaverMock: &goodSaverMock{
				name:      "Apple",
				projectID: "1",
				err:       storageErr,
			},
			reqBody:    `{"name":"Apple"}`,
			projectID:  "1",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"status":"Error","error":"failed to save good"}`,
		},
		{
			name:       "Missing projectId",
			reqBody:    `{"name":"Apple"}`,
			projectID:  "",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"status":"Error","error":"invalid url params"}`,
		},
		{
			name: "Coulnd not invalidate cache",
			goodSaverMock: &goodSaverMock{
				name:      "Apple",
				projectID: "1",
				resp: &models.Good{
					ID:          1,
					ProjectID:   1,
					Name:        "Apple",
					Description: "NO DESC",
					Priority:    5,
					Removed:     false,
					CreatedAt:   time.UnixMilli(1234567890),
				},
			},
			invalidCacheMock: &invalidCacheMock{
				err: storageErr,
			},
			reqBody:    `{"name":"Apple"}`,
			projectID:  "1",
			wantBody:   `{"status":"Error","error":"failed to invalid cached list"}`,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			goodSaverMock := mocks.NewMockGoodSaver(ctrl)
			producerMock := producer_mocks.NewMockProducerInterface(ctrl)

			if tc.goodSaverMock != nil {
				goodSaverMock.EXPECT().
					SaveGood(gomock.Any(), tc.goodSaverMock.name, tc.goodSaverMock.projectID).
					Return(tc.goodSaverMock.resp, tc.goodSaverMock.err).Times(1)
			}

			if tc.invalidCacheMock != nil {
				goodSaverMock.EXPECT().
					InvalidList(gomock.Any()).
					Return(tc.invalidCacheMock.err).Times(1)
			}

			if tc.producerMock != nil {
				producerMock.EXPECT().
					SendAsync(gomock.Any()).
					Return(tc.producerMock.err).Times(1)
			}

			handler := create.New(slogdiscard.NewDiscardLogger(), goodSaverMock, producerMock)

			url := "/good/create"
			if tc.projectID != "" {
				url = fmt.Sprintf("/good/create?projectId=%s", tc.projectID)
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(tc.reqBody)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, rr.Code, tc.wantStatus, "incorrect status code")
			require.JSONEq(t, tc.wantBody, rr.Body.String(), "mismatch response body")
		})
	}
}
