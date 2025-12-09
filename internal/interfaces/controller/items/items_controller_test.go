package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
	"Aicon-assignment/internal/usecase"
)

// MockItemUsecase はテスト用のモックユースケース
type MockItemUsecase struct {
	mock.Mock
}

func (m *MockItemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.Item), args.Error(1)
}

func (m *MockItemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Item), args.Error(1)
}

func (m *MockItemUsecase) CreateItem(ctx context.Context, input usecase.CreateItemInput) (*entity.Item, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Item), args.Error(1)
}

func (m *MockItemUsecase) UpdateItem(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error) {
	args := m.Called(ctx, id, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Item), args.Error(1)
}

func (m *MockItemUsecase) DeleteItem(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockItemUsecase) GetCategorySummary(ctx context.Context) (*usecase.CategorySummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CategorySummary), args.Error(1)
}

func TestItemHandler_UpdateItem(t *testing.T) {
	tests := []struct {
		name           string
		itemID         string
		requestBody    string
		setupMock      func(*MockItemUsecase)
		expectedStatus int
		checkResponse  func(*testing.T, string)
	}{
		{
			name:        "正常系: 複数フィールド同時更新",
			itemID:      "1",
			requestBody: `{"name": "新しい名前", "brand": "新しいブランド", "purchase_price": 1500000}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				updatedItem, _ := entity.NewItem("新しい名前", "時計", "新しいブランド", 1500000, "2023-01-01")
				updatedItem.ID = 1
				input := usecase.UpdateItemInput{
					Name:          strPtr("新しい名前"),
					Brand:         strPtr("新しいブランド"),
					PurchasePrice: intPtr(1500000),
				}
				mockUsecase.On("UpdateItem", mock.Anything, int64(1), input).Return(updatedItem, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var item entity.Item
				err := json.Unmarshal([]byte(body), &item)
				assert.NoError(t, err)
				assert.Equal(t, "新しい名前", item.Name)
				assert.Equal(t, "新しいブランド", item.Brand)
				assert.Equal(t, 1500000, item.PurchasePrice)
			},
		},
		{
			name:        "異常系: 無効な ID",
			itemID:      "invalid",
			requestBody: `{"name": "名前"}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				// UpdateItemは呼ばれない
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "invalid item ID", errResp.Error)
			},
		},
		{
			name:        "異常系: 更新フィールドがない",
			itemID:      "1",
			requestBody: `{}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				// UpdateItemは呼ばれない
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "at least one field must be provided for update", errResp.Error)
			},
		},
		{
			name:        "異常系: アイテムが見つからない (404)",
			itemID:      "999",
			requestBody: `{"name": "名前"}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				input := usecase.UpdateItemInput{
					Name: strPtr("名前"),
				}
				mockUsecase.On("UpdateItem", mock.Anything, int64(999), input).Return((*entity.Item)(nil), domainErrors.ErrItemNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "item not found", errResp.Error)
			},
		},
		{
			name:        "異常系: バリデーションエラー (400)",
			itemID:      "1",
			requestBody: `{"name": ""}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				input := usecase.UpdateItemInput{
					Name: strPtr(""),
				}
				mockUsecase.On("UpdateItem", mock.Anything, int64(1), input).Return((*entity.Item)(nil), domainErrors.ErrInvalidInput)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "validation failed", errResp.Error)
			},
		},
		{
			name:        "異常系: 内部エラー (500)",
			itemID:      "1",
			requestBody: `{"name": "名前"}`,
			setupMock: func(mockUsecase *MockItemUsecase) {
				input := usecase.UpdateItemInput{
					Name: strPtr("名前"),
				}
				mockUsecase.On("UpdateItem", mock.Anything, int64(1), input).Return((*entity.Item)(nil), domainErrors.ErrDatabaseError)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				var errResp ErrorResponse
				err := json.Unmarshal([]byte(body), &errResp)
				assert.NoError(t, err)
				assert.Equal(t, "failed to update item", errResp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Echoインスタンスとモックの準備
			e := echo.New()
			mockUsecase := new(MockItemUsecase)
			tt.setupMock(mockUsecase)
			handler := NewItemHandler(mockUsecase)

			// リクエストの作成
			req := httptest.NewRequest(http.MethodPatch, "/items/"+tt.itemID, strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/items/:id")
			c.SetParamNames("id")
			c.SetParamValues(tt.itemID)

			// ハンドラーの実行
			err := handler.UpdateItem(c)

			// アサーション
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}

			mockUsecase.AssertExpectations(t)
		})
	}
}

// ヘルパー関数
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
