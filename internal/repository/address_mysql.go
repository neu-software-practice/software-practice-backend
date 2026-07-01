package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/neuhis/software-practice-backend/internal/model"
)

type addressMySQLRepo struct {
	db *sql.DB
}

// NewAddressRepository creates a new MySQL-based AddressRepository.
func NewAddressRepository(db *sql.DB) AddressRepository {
	return &addressMySQLRepo{db: db}
}

func (r *addressMySQLRepo) Create(ctx context.Context, addr *model.Address) error {
	touchTimestamps(&addr.CreatedAt, &addr.UpdatedAt)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO addresses (id, patient_id, name, phone, province, city, district, detail, is_default, tag, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		addr.ID, addr.PatientID, addr.Name, addr.Phone,
		addr.Province, addr.City, addr.District, addr.Detail,
		addr.IsDefault, addr.Tag, addr.CreatedAt, addr.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create address: %w", err)
	}
	return nil
}

func (r *addressMySQLRepo) FindByID(ctx context.Context, id string) (*model.Address, error) {
	addr := &model.Address{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, patient_id, name, phone, province, city, district, detail, is_default, tag, created_at, updated_at
		FROM addresses WHERE id = ?`, id,
	).Scan(&addr.ID, &addr.PatientID, &addr.Name, &addr.Phone,
		&addr.Province, &addr.City, &addr.District, &addr.Detail,
		&addr.IsDefault, &addr.Tag, &addr.CreatedAt, &addr.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrAddressNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find address by id: %w", err)
	}
	return addr, nil
}

func (r *addressMySQLRepo) ListByPatient(ctx context.Context, patientID string) ([]model.Address, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, patient_id, name, phone, province, city, district, detail, is_default, tag, created_at, updated_at
		FROM addresses WHERE patient_id = ? ORDER BY is_default DESC, created_at ASC`,
		patientID,
	)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var addrs []model.Address
	for rows.Next() {
		var a model.Address
		if err := rows.Scan(&a.ID, &a.PatientID, &a.Name, &a.Phone,
			&a.Province, &a.City, &a.District, &a.Detail,
			&a.IsDefault, &a.Tag, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan address: %w", err)
		}
		addrs = append(addrs, a)
	}
	if addrs == nil {
		addrs = []model.Address{}
	}
	return addrs, nil
}

func (r *addressMySQLRepo) CountByPatient(ctx context.Context, patientID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM addresses WHERE patient_id = ?`, patientID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count addresses: %w", err)
	}
	return count, nil
}

func (r *addressMySQLRepo) Update(ctx context.Context, addr *model.Address) error {
	addr.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE addresses SET name=?, phone=?, province=?, city=?, district=?, detail=?, is_default=?, tag=?, updated_at=? WHERE id=?`,
		addr.Name, addr.Phone, addr.Province, addr.City, addr.District, addr.Detail,
		addr.IsDefault, addr.Tag, addr.UpdatedAt, addr.ID,
	)
	if err != nil {
		return fmt.Errorf("update address: %w", err)
	}
	return nil
}

func (r *addressMySQLRepo) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM addresses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrAddressNotFound
	}
	return nil
}

func (r *addressMySQLRepo) ClearDefaultByPatient(ctx context.Context, patientID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE addresses SET is_default = 0 WHERE patient_id = ? AND is_default = 1`,
		patientID,
	)
	if err != nil {
		return fmt.Errorf("clear default addresses: %w", err)
	}
	return nil
}

func (r *addressMySQLRepo) SetDefault(ctx context.Context, id string, patientID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`UPDATE addresses SET is_default = 0 WHERE patient_id = ? AND is_default = 1`,
		patientID,
	)
	if err != nil {
		return fmt.Errorf("clear defaults in tx: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE addresses SET is_default = 1 WHERE id = ? AND patient_id = ?`,
		id, patientID,
	)
	if err != nil {
		return fmt.Errorf("set default in tx: %w", err)
	}

	return tx.Commit()
}
