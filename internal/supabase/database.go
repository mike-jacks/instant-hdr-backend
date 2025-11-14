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

func (d *DatabaseClient) CreateProject(userID uuid.UUID, imagenProjectUUID string, metadata map[string]interface{}) (*models.Project, error) {
	metadataJSON, _ := json.Marshal(metadata)
	
	var project models.Project
	err := d.db.QueryRow(`
		INSERT INTO projects (user_id, imagen_project_uuid, status, metadata)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, imagen_project_uuid, status, progress, profile_key, edit_id, metadata, error_message, created_at, updated_at
	`, userID, imagenProjectUUID, "created", metadataJSON).Scan(
		&project.ID, &project.UserID, &project.ImagenProjectUUID, &project.Status,
		&project.Progress, &project.ProfileKey, &project.EditID, &project.Metadata,
		&project.ErrorMessage, &project.CreatedAt, &project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return &project, nil
}

func (d *DatabaseClient) GetProject(projectID, userID uuid.UUID) (*models.Project, error) {
	var project models.Project
	err := d.db.QueryRow(`
		SELECT id, user_id, imagen_project_uuid, status, progress, profile_key, edit_id, metadata, error_message, created_at, updated_at
		FROM projects
		WHERE id = $1 AND user_id = $2
	`, projectID, userID).Scan(
		&project.ID, &project.UserID, &project.ImagenProjectUUID, &project.Status,
		&project.Progress, &project.ProfileKey, &project.EditID, &project.Metadata,
		&project.ErrorMessage, &project.CreatedAt, &project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

func (d *DatabaseClient) ListProjects(userID uuid.UUID) ([]models.Project, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, imagen_project_uuid, status, progress, profile_key, edit_id, metadata, error_message, created_at, updated_at
		FROM projects
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		err := rows.Scan(
			&project.ID, &project.UserID, &project.ImagenProjectUUID, &project.Status,
			&project.Progress, &project.ProfileKey, &project.EditID, &project.Metadata,
			&project.ErrorMessage, &project.CreatedAt, &project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (d *DatabaseClient) UpdateProjectStatus(projectID uuid.UUID, status string, progress int) error {
	_, err := d.db.Exec(`
		UPDATE projects
		SET status = $1, progress = $2
		WHERE id = $3
	`, status, progress, projectID)
	return err
}

func (d *DatabaseClient) UpdateProjectEditID(projectID uuid.UUID, editID string) error {
	_, err := d.db.Exec(`
		UPDATE projects
		SET edit_id = $1, status = 'processing'
		WHERE id = $2
	`, editID, projectID)
	return err
}

func (d *DatabaseClient) UpdateProjectError(projectID uuid.UUID, errorMsg string) error {
	_, err := d.db.Exec(`
		UPDATE projects
		SET status = 'failed', error_message = $1
		WHERE id = $2
	`, errorMsg, projectID)
	return err
}

func (d *DatabaseClient) DeleteProject(projectID, userID uuid.UUID) error {
	_, err := d.db.Exec(`
		DELETE FROM projects
		WHERE id = $1 AND user_id = $2
	`, projectID, userID)
	return err
}

func (d *DatabaseClient) CreateProjectFile(file *models.ProjectFile) error {
	_, err := d.db.Exec(`
		INSERT INTO project_files (project_id, user_id, filename, imagen_file_id, storage_path, storage_url, file_size, mime_type, is_final)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, file.ProjectID, file.UserID, file.Filename, file.ImagenFileID, file.StoragePath,
		file.StorageURL, file.FileSize, file.MimeType, file.IsFinal)
	return err
}

func (d *DatabaseClient) GetProjectFiles(projectID, userID uuid.UUID) ([]models.ProjectFile, error) {
	rows, err := d.db.Query(`
		SELECT id, project_id, user_id, filename, imagen_file_id, storage_path, storage_url, file_size, mime_type, is_final, created_at
		FROM project_files
		WHERE project_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project files: %w", err)
	}
	defer rows.Close()

	var files []models.ProjectFile
	for rows.Next() {
		var file models.ProjectFile
		err := rows.Scan(
			&file.ID, &file.ProjectID, &file.UserID, &file.Filename,
			&file.ImagenFileID, &file.StoragePath, &file.StorageURL,
			&file.FileSize, &file.MimeType, &file.IsFinal, &file.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

func (d *DatabaseClient) GetProjectByImagenUUID(imagenProjectUUID string) (*models.Project, error) {
	var project models.Project
	err := d.db.QueryRow(`
		SELECT id, user_id, imagen_project_uuid, status, progress, profile_key, edit_id, metadata, error_message, created_at, updated_at
		FROM projects
		WHERE imagen_project_uuid = $1
	`, imagenProjectUUID).Scan(
		&project.ID, &project.UserID, &project.ImagenProjectUUID, &project.Status,
		&project.Progress, &project.ProfileKey, &project.EditID, &project.Metadata,
		&project.ErrorMessage, &project.CreatedAt, &project.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

func (d *DatabaseClient) Close() error {
	return d.db.Close()
}

