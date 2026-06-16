// Package repository is the data-access layer (Repository Pattern, PLAN §2.1).
// Each aggregate exposes an interface plus a GORM-backed implementation so the
// service layer can depend on the interface and be unit-tested with mocks.
//
// Atomic, multi-repository operations (charging, dispensing) use TxManager: it
// opens a GORM transaction and threads the *gorm.DB through context.Context, so
// any repository call made inside the callback automatically joins the tx.
package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// ErrNotFound is the layer-neutral "no row" error. Services translate it into
// the appropriate apperr so the rest of the code never imports gorm.
var ErrNotFound = errors.New("repository: record not found")

// wrapNotFound normalizes gorm's not-found sentinel.
func wrapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

type txKey struct{}

// base is embedded by every repository to resolve the active connection: the
// transaction bound to ctx if any, otherwise the shared pool.
type base struct{ db *gorm.DB }

func (b base) conn(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return b.db.WithContext(ctx)
}

// TxManager runs a function inside a single database transaction.
type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct{ db *gorm.DB }

// NewTxManager builds a GORM-backed TxManager.
func NewTxManager(db *gorm.DB) TxManager { return &txManager{db: db} }

func (m *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Page holds pagination input. Zero values fall back to sensible defaults.
type Page struct {
	Page  int
	Limit int
}

// Normalized clamps page/limit into valid ranges (page≥1, 1≤limit≤100).
func (p Page) Normalized() (page, limit int) {
	page, limit = p.Page, p.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// apply adds OFFSET/LIMIT to a query.
func (p Page) apply(db *gorm.DB) *gorm.DB {
	page, limit := p.Normalized()
	return db.Offset((page - 1) * limit).Limit(limit)
}
