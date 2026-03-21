package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Product mirrors the products table.
type Product struct {
	ID            string
	WorkspaceID   string
	SKU           string
	Name          string
	Category      string
	DefaultFields map[string]*string
	PrintCount    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ProductFilter holds query parameters for listing products.
type ProductFilter struct {
	WorkspaceID string
	Query       string
	Category    string
	Page        int
	PerPage     int
}

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

// Create inserts a new product.
func (r *ProductRepo) Create(ctx context.Context, workspaceID, sku, name, category string, defaultFields map[string]*string) (*Product, error) {
	fieldsJSON, _ := json.Marshal(defaultFields)
	var p Product
	err := r.db.QueryRow(ctx, `
		INSERT INTO products (workspace_id, sku, name, category, default_fields)
		VALUES ($1, NULLIF($2,''), $3, NULLIF($4,''), $5)
		RETURNING id, workspace_id, COALESCE(sku,''), name, COALESCE(category,''),
		          COALESCE(default_fields,'{}')::text, print_count, created_at, updated_at
	`, workspaceID, sku, name, category, fieldsJSON).Scan(
		&p.ID, &p.WorkspaceID, &p.SKU, &p.Name, &p.Category,
		new(string), &p.PrintCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// Re-scan default_fields separately to unmarshal JSON
	var raw string
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(default_fields,'{}')::text FROM products WHERE id = $1`, p.ID).Scan(&raw)
	if raw != "" && raw != "null" {
		_ = json.Unmarshal([]byte(raw), &p.DefaultFields)
	}
	return &p, nil
}

// GetByID fetches a product, checking workspace ownership.
func (r *ProductRepo) GetByID(ctx context.Context, id, workspaceID string) (*Product, error) {
	return r.scan(r.db.QueryRow(ctx, `
		SELECT id, workspace_id, COALESCE(sku,''), name, COALESCE(category,''),
		       COALESCE(default_fields,'{}')::text, print_count, created_at, updated_at
		FROM products
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID))
}

// List returns paginated products with optional search + category filter.
func (r *ProductRepo) List(ctx context.Context, f ProductFilter) ([]Product, int, error) {
	if f.PerPage <= 0 {
		f.PerPage = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PerPage

	args := []any{f.WorkspaceID}
	where := "workspace_id = $1"

	if f.Category != "" {
		args = append(args, f.Category)
		where += " AND category = $" + strconv.Itoa(len(args))
	}
	if f.Query != "" {
		args = append(args, "%"+f.Query+"%")
		where += " AND name ILIKE $" + strconv.Itoa(len(args))
	}

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM products WHERE "+where, args...).Scan(&total)

	args = append(args, f.PerPage, offset)
	rows, err := r.db.Query(ctx, `
		SELECT id, workspace_id, COALESCE(sku,''), name, COALESCE(category,''),
		       COALESCE(default_fields,'{}')::text, print_count, created_at, updated_at
		FROM products
		WHERE `+where+`
		ORDER BY name ASC
		LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		p, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, *p)
	}
	return products, total, rows.Err()
}

// UpdateFields merges new fields into default_fields and optionally updates metadata.
func (r *ProductRepo) UpdateFields(ctx context.Context, id, workspaceID string, fields map[string]*string, name, category string) error {
	fieldsJSON, _ := json.Marshal(fields)
	_, err := r.db.Exec(ctx, `
		UPDATE products
		SET default_fields = COALESCE(default_fields,'{}') || $3::jsonb,
		    name           = CASE WHEN $4 != '' THEN $4 ELSE name END,
		    category       = CASE WHEN $5 != '' THEN $5 ELSE category END,
		    updated_at     = NOW()
		WHERE id = $1 AND workspace_id = $2
	`, id, workspaceID, fieldsJSON, name, category)
	return err
}

// IncrementPrintCount atomically bumps the print counter.
func (r *ProductRepo) IncrementPrintCount(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE products SET print_count = print_count + 1, updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (r *ProductRepo) scan(row scanner) (*Product, error) {
	return r.scanRow(row)
}

func (r *ProductRepo) scanRow(row scanner) (*Product, error) {
	var p Product
	var fieldsRaw string
	err := row.Scan(
		&p.ID, &p.WorkspaceID, &p.SKU, &p.Name, &p.Category,
		&fieldsRaw, &p.PrintCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if fieldsRaw != "" && fieldsRaw != "null" {
		_ = json.Unmarshal([]byte(fieldsRaw), &p.DefaultFields)
	}
	return &p, nil
}
