package service

import (
	"context"
	"fmt"
	db "github/heimaolst/urlshorter/db/sqlc"
	"github/heimaolst/urlshorter/internal/model"
	"github/heimaolst/urlshorter/internal/util"
	"time"
)

type URLService struct {
	querier            db.Queries
	shortCodeGenerator ShortCodeGenerator
}

func (s *URLService) CreateURL(ctx context.Context, req model.CreateURLRequest) (*model.CreateURLResponse, error) {

	var shortCode string
	var is_custom bool
	var expireAt time.Time

	if req.CustomCode != "" {
		isAvailable, err := s.querier.IsShortCodeAvailable(ctx, req.CustomCode)
		if err != nil {
			return nil, err
		}

		if !isAvailable {
			return nil, fmt.Errorf("custom code %s had been used", req.CustomCode)
		}

		shortCode = req.CustomCode
		is_custom = true
	} else {
		var err error
		shortCode, err = util.GenerateShortCode(6)
		if err != nil {
			return nil, err
		}
		is_custom = false
	}
	if req.Duration != nil {
		expireAt = time.Now().Add(time.Duration(*req.Duration) * time.Hour)
	} else {
		expireAt = time.Now().Add(24 * time.Hour)
	}

	//Insert DB
	s.querier.CreateUrl(ctx, db.CreateUrlParams{
		OriginalUrl: req.OriginalURL,
		ShortCode:   shortCode,
		IsCustom:    is_custom,
		ExpiredAt:   expireAt,
	})
}
