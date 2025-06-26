package model

import "time"

type CreateURLRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	CustomCode  string `json:"custom_code,omitempty" binding:"omitempty,min=6,max=10,alphanum"`
	Duration    *int   `json:"duration,omitempty" binding:"omitempty,min=1,max=100"`
}

type CreateURLResponse struct {
	Success   bool      `json:"success"`
	ShortCode string    `json:"short_code"`
	ExpireAt  time.Time `json:"expire_at"`
}
type URL struct {
	ShortCode   string    `json:"short_code redis:"-"`
	OriginalURL string    `json:"original_url" redis:"original_url"`
	ExpiredAt   time.Time `json:"expired_at" redis:"expired_at"`
	IsCustom    bool      `json:"is_custom" redis:"is_custom"`
	Clicks      int       `json:"clicks" redis: "clicks"`
}
