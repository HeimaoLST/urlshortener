package model

import "time"

type CreateURLRequest struct {
	OriginalURL string `json:"original_url" validate:"required,url"`
	CustomCode  string `json:"custom_code,omitempty" validate:"omitempty,min=6,max=10,alphanum"`
	Duration    *int   `json:"duration,omitempty" validate:"omitempty,min=1,max=100"`
}

type CreateURLResponse struct {
	Success   bool      `json:"success"`
	ShortCode string    `json:"short_code"`
	ExpireAt  time.Time `json:"expire_at"`
}
type URL struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	ExpiredAt   time.Time `json:"expired_at"`
	IsCustom    bool      `json:"is_custom"`
}
