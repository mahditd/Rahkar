package application

import (
	"context"

	"rahcar/backend/internal/domain"
)

type Market interface {
	Overview(context.Context) (domain.MarketSnapshot, error)
}
