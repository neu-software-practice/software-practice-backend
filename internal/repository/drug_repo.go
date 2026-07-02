package repository

import (
	"context"

	"github.com/neuhis/software-practice-backend/internal/model"
)

// DrugRepository defines the pharmacy catalog and stock interface.
type DrugRepository interface {
	FindEnabledByNameOrAlias(ctx context.Context, name string) (*model.Drug, error)
	DecrementStock(ctx context.Context, name string, quantity int) error
}
