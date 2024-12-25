package database

import (
	"database/sql"
	"errors"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
)

type Video struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ThumbnailURL *string   `json:"thumbnail_url"`
	VideoURL     *string   `json:"video_url"`
	CreateVideoParams
}

type CreateVideoParams struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	UserID      uuid.UUID `json:"user_id"`
}

func (c Client) GetVideos(userID uuid.UUID) ([]Video, error) {
	query := `
	SELECT
		id,
		created_at,
		updated_at,
		title,
		description,
		thumbnail_url,
		video_url,
		user_id
	FROM videos
	WHERE user_id = ?
	ORDER BY created_at DESC
	`

	rows, err := c.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []Video{}
	for rows.Next() {
		var video Video
		if err := rows.Scan(
			&video.ID,
			&video.CreatedAt,
			&video.UpdatedAt,
			&video.Title,
			&video.Description,
			&video.ThumbnailURL,
			&video.VideoURL,
			&video.UserID,
		); err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func (c Client) CreateVideo(params CreateVideoParams) (Video, error) {
	id := uuid.New()
	query := `
	INSERT INTO videos (
		id,
		created_at,
		updated_at,
		title,
		description,
		user_id
	) VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?)
	`
	_, err := c.db.Exec(query, id, params.Title, params.Description, params.UserID)
	if err != nil {
		return Video{}, err
	}

	return c.GetVideo(id)
}

func (c Client) GetVideo(id uuid.UUID) (Video, error) {
	query := `
	SELECT
		id,
		created_at,
		updated_at,
		title,
		description,
		thumbnail_url,
		video_url,
		user_id
	FROM videos
	WHERE id = ?
	`

	var video Video
	err := c.db.QueryRow(query, id).Scan(
		&video.ID,
		&video.CreatedAt,
		&video.UpdatedAt,
		&video.Title,
		&video.Description,
		&video.ThumbnailURL,
		&video.VideoURL,
		&video.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Video{}, nil
		}
		return Video{}, err
	}

	return video, nil
}

func (c Client) UpdateVideo(video Video) error {
	query := `
	UPDATE videos
	SET
		title = ?,
		description = ?,
		thumbnail_url = ?,
		video_url = ?,
		user_id = ?
	WHERE id = ?
	`

	if video.ThumbnailURL == nil {
		return errors.New("thumbnail URL is required")
	}

	_, err := c.db.Exec(
		query,
		video.Title,
		video.Description,
		*video.ThumbnailURL,
		video.VideoURL,
		video.UserID,
		video.ID,
	)
	return err
}

func (c Client) DeleteVideo(id uuid.UUID) error {
	query := `
	DELETE FROM videos
	WHERE id = ?
	`
	_, err := c.db.Exec(query, id)
	return err
}

func readImageFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getFileExtension(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	default:
		return "bin"
	}
}
