package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	dbmodel "github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	dbmodel.DB = db
	dbmodel.LOG_DB = db
	if err := db.AutoMigrate(&dbmodel.CallMetadata{}); err != nil {
		t.Fatalf("failed to migrate call_metadata: %v", err)
	}
	return func() {
		dbmodel.DB = nil
		dbmodel.LOG_DB = nil
	}
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request = req
	return c, w
}

func TestLogCallMetadataSuccess(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	c, _ := newTestContext()
	c.Set(helper.RequestIdKey, "req-1")
	c.Set(ctxkey.Id, 42)
	c.Set(ctxkey.TokenId, 7)
	c.Set(ctxkey.ChannelId, 3)
	c.Set(ctxkey.OriginalModel, "gpt-4o")
	c.Set("is_stream", true)
	c.Set("prompt_tokens", 11)
	c.Set("completion_tokens", 22)

	logCallMetadata(c, "gen-1", 0, 123, nil)

	var rows []dbmodel.CallMetadata
	if err := dbmodel.DB.Find(&rows).Error; err != nil {
		t.Fatalf("query call_metadata: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.GenerationID != "gen-1" || row.RequestID != "req-1" {
		t.Fatalf("unexpected ids: %+v", row)
	}
	if row.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", row.StatusCode)
	}
	if row.LatencyMs != 123 || row.Attempt != 0 {
		t.Fatalf("latency/attempt mismatch: %+v", row)
	}
	if row.PromptTokens != 11 || row.CompletionTokens != 22 || !row.IsStream {
		t.Fatalf("token/stream mismatch: %+v", row)
	}
}

func TestLogCallMetadataError(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	c, _ := newTestContext()
	c.Set(helper.RequestIdKey, "req-2")
	c.Set(ctxkey.Id, 99)
	c.Set(ctxkey.TokenId, 8)
	c.Set(ctxkey.ChannelId, 4)
	c.Set(ctxkey.OriginalModel, "gpt-3.5")

	errPayload := &relaymodel.ErrorWithStatusCode{
		StatusCode: http.StatusTooManyRequests,
		Error: relaymodel.Error{
			Code: "rate_limit",
		},
	}

	logCallMetadata(c, "gen-2", 1, 456, errPayload)

	var rows []dbmodel.CallMetadata
	if err := dbmodel.DB.Find(&rows).Error; err != nil {
		t.Fatalf("query call_metadata: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", row.StatusCode)
	}
	if row.Attempt != 1 || row.LatencyMs != 456 {
		t.Fatalf("latency/attempt mismatch: %+v", row)
	}
	if row.UserID != 99 || row.ChannelID != 4 || row.TokenID != 8 {
		t.Fatalf("id fields mismatch: %+v", row)
	}
}
