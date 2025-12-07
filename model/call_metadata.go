package model

// CallMetadata stores per-request metadata for routing and auditing.
type CallMetadata struct {
	ID               int    `json:"id" gorm:"primaryKey"`
	GenerationID     string `json:"generation_id" gorm:"index"`
	RequestID        string `json:"request_id" gorm:"index"`
	UserID           int    `json:"user_id" gorm:"index"`
	TokenID          int    `json:"token_id" gorm:"index"`
	ChannelID        int    `json:"channel_id" gorm:"index"`
	Model            string `json:"model" gorm:"index"`
	APIPath          string `json:"api_path"`
	IsStream         bool   `json:"is_stream"`
	StatusCode       int    `json:"status_code"`
	LatencyMs        int64  `json:"latency_ms"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Attempt          int    `json:"attempt"`
	CreatedAt        int64  `json:"created_at" gorm:"autoCreateTime:milli"`
}

// InsertCallMetadata persists one metadata record; caller should best-effort log and continue on error.
func InsertCallMetadata(meta *CallMetadata) error {
	return DB.Create(meta).Error
}
