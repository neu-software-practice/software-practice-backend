package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type drugMySQLRepo struct {
	db *sql.DB
}

// NewDrugRepository creates a MySQL-based drug catalog repository.
func NewDrugRepository(db *sql.DB) DrugRepository {
	return &drugMySQLRepo{db: db}
}

func (r *drugMySQLRepo) FindEnabledByNameOrAlias(ctx context.Context, name string) (*model.Drug, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, aliases, spec, default_dosage, default_days, unit_price, stock_quantity, enabled, created_at, updated_at
		FROM drugs
		WHERE enabled = 1 AND (name = ? OR JSON_CONTAINS(aliases, JSON_QUOTE(?)))
		ORDER BY CASE WHEN name = ? THEN 0 ELSE 1 END, name ASC
		LIMIT 1`, name, name, name,
	)
	if err != nil {
		return nil, fmt.Errorf("find drug by name or alias: %w", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, model.ErrDrugNotFound
	}
	drug, err := scanDrug(rows)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate drug rows: %w", err)
	}
	return drug, nil
}

func (r *drugMySQLRepo) DecrementStock(ctx context.Context, name string, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("%w: quantity must be positive", model.ErrValidation)
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE drugs
		SET stock_quantity = stock_quantity - ?
		WHERE enabled = 1 AND name = ? AND stock_quantity >= ?`,
		quantity, name, quantity,
	)
	if err != nil {
		return fmt.Errorf("decrement drug stock: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read stock update result: %w", err)
	}
	if affected > 0 {
		return nil
	}

	drug, findErr := r.FindEnabledByNameOrAlias(ctx, name)
	if findErr != nil {
		return findErr
	}
	if drug.StockQuantity < quantity {
		return fmt.Errorf("%w: %s", model.ErrDrugStockInsufficient, name)
	}
	return fmt.Errorf("%w: %s", model.ErrDrugNotFound, name)
}

func scanDrug(scanner rowScanner) (*model.Drug, error) {
	var (
		drug       model.Drug
		aliasesRaw []byte
	)
	err := scanner.Scan(
		&drug.ID, &drug.Name, &aliasesRaw, &drug.Spec, &drug.DefaultDosage,
		&drug.DefaultDays, &drug.UnitPrice, &drug.StockQuantity, &drug.Enabled,
		&drug.CreatedAt, &drug.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrDrugNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan drug: %w", err)
	}
	if len(aliasesRaw) > 0 {
		if err := json.Unmarshal(aliasesRaw, &drug.Aliases); err != nil {
			return nil, fmt.Errorf("unmarshal drug aliases: %w", err)
		}
	}
	if drug.Aliases == nil {
		drug.Aliases = []string{}
	}
	return &drug, nil
}
