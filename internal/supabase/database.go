package supabase

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"instant-hdr-backend/internal/models"
)

type DatabaseClient struct {
	db *sql.DB
}

func NewDatabaseClient(connectionString string) (*DatabaseClient, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseClient{db: db}, nil
}

func (d *DatabaseClient) CreateOrder(orderID, userID uuid.UUID, metadata map[string]interface{}) (*models.Order, error) {
	metadataJSON, _ := json.Marshal(metadata)

	var order models.Order
	err := d.db.QueryRow(`
		INSERT INTO orders (id, user_id, status, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, status, progress, metadata, error_message, created_at, updated_at
	`, orderID, userID, "created", metadataJSON).Scan(
		&order.ID, &order.UserID, &order.Status,
		&order.Progress, &order.Metadata, &order.ErrorMessage, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return &order, nil
}

func (d *DatabaseClient) GetOrder(orderID, userID uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := d.db.QueryRow(`
		SELECT id, user_id, status, progress, metadata, error_message, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID).Scan(
		&order.ID, &order.UserID, &order.Status,
		&order.Progress, &order.Metadata, &order.ErrorMessage, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (d *DatabaseClient) ListOrders(userID uuid.UUID) ([]models.Order, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, status, progress, metadata, error_message, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Status,
			&order.Progress, &order.Metadata, &order.ErrorMessage, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (d *DatabaseClient) UpdateOrderStatus(orderID uuid.UUID, status string, progress int) error {
	_, err := d.db.Exec(`
		UPDATE orders
		SET status = $1, progress = $2
		WHERE id = $3
	`, status, progress, orderID)
	return err
}

func (d *DatabaseClient) UpdateOrderError(orderID uuid.UUID, errorMsg string) error {
	_, err := d.db.Exec(`
		UPDATE orders
		SET status = 'failed', error_message = $1
		WHERE id = $2
	`, errorMsg, orderID)
	return err
}

func (d *DatabaseClient) DeleteOrder(orderID, userID uuid.UUID) error {
	_, err := d.db.Exec(`
		DELETE FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID)
	return err
}

func (d *DatabaseClient) GetOrderByAutoEnhanceOrderID(autoenhanceOrderID string) (*models.Order, error) {
	// Parse the order_id string as UUID and query by id
	orderID, err := uuid.Parse(autoenhanceOrderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order id: %w", err)
	}
	
	// Query by id (no userID check since this is used for webhooks)
	var order models.Order
	err = d.db.QueryRow(`
		SELECT id, user_id, status, progress, metadata, error_message, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, orderID).Scan(
		&order.ID, &order.UserID, &order.Status,
		&order.Progress, &order.Metadata, &order.ErrorMessage, &order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

func (d *DatabaseClient) CreateOrderFile(file *models.OrderFile) error {
	_, err := d.db.Exec(`
		INSERT INTO order_files (order_id, user_id, filename, autoenhance_image_id, storage_path, storage_url, file_size, mime_type, is_final)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, file.OrderID, file.UserID, file.Filename, file.AutoEnhanceImageID, file.StoragePath,
		file.StorageURL, file.FileSize, file.MimeType, file.IsFinal)
	return err
}

func (d *DatabaseClient) GetOrderFiles(orderID, userID uuid.UUID) ([]models.OrderFile, error) {
	rows, err := d.db.Query(`
		SELECT id, order_id, user_id, filename, autoenhance_image_id, storage_path, storage_url, file_size, mime_type, is_final, created_at
		FROM order_files
		WHERE order_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`, orderID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order files: %w", err)
	}
	defer rows.Close()

	var files []models.OrderFile
	for rows.Next() {
		var file models.OrderFile
		err := rows.Scan(
			&file.ID, &file.OrderID, &file.UserID, &file.Filename,
			&file.AutoEnhanceImageID, &file.StoragePath, &file.StorageURL,
			&file.FileSize, &file.MimeType, &file.IsFinal, &file.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

func (d *DatabaseClient) DeleteOrderFile(fileID uuid.UUID) error {
	_, err := d.db.Exec(`
		DELETE FROM order_files
		WHERE id = $1
	`, fileID)
	return err
}

func (d *DatabaseClient) CreateBracket(bracket *models.Bracket) error {
	_, err := d.db.Exec(`
		INSERT INTO brackets (order_id, bracket_id, image_id, filename, upload_url, is_uploaded, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, bracket.OrderID, bracket.BracketID, bracket.ImageID, bracket.Filename,
		bracket.UploadURL, bracket.IsUploaded, bracket.Metadata)
	return err
}

func (d *DatabaseClient) GetBracketsByOrderID(orderID uuid.UUID) ([]models.Bracket, error) {
	rows, err := d.db.Query(`
		SELECT id, order_id, bracket_id, image_id, filename, upload_url, is_uploaded, metadata, created_at
		FROM brackets
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get brackets: %w", err)
	}
	defer rows.Close()

	var brackets []models.Bracket
	for rows.Next() {
		var bracket models.Bracket
		err := rows.Scan(
			&bracket.ID, &bracket.OrderID, &bracket.BracketID, &bracket.ImageID,
			&bracket.Filename, &bracket.UploadURL, &bracket.IsUploaded,
			&bracket.Metadata, &bracket.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bracket: %w", err)
		}
		brackets = append(brackets, bracket)
	}

	return brackets, nil
}

func (d *DatabaseClient) UpdateBracketImageID(bracketID string, imageID string) error {
	_, err := d.db.Exec(`
		UPDATE brackets
		SET image_id = $1
		WHERE bracket_id = $2
	`, imageID, bracketID)
	return err
}

func (d *DatabaseClient) Close() error {
	return d.db.Close()
}
