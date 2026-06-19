package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"bim-viewer/internal/model"
)

func (r *PostgresRepo) MigrateAnnotations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS annotations (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
			type VARCHAR(16) NOT NULL DEFAULT 'element',
			element_id VARCHAR(64),
			position DOUBLE PRECISION[3] NOT NULL,
			title VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			priority VARCHAR(16) NOT NULL DEFAULT 'normal',
			status VARCHAR(16) NOT NULL DEFAULT 'open',
			creator VARCHAR(128) NOT NULL DEFAULT 'anonymous',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS annotation_comments (
			id VARCHAR(64) PRIMARY KEY,
			annotation_id VARCHAR(64) NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			author VARCHAR(128) NOT NULL DEFAULT 'anonymous',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS annotation_attachments (
			id VARCHAR(64) PRIMARY KEY,
			owner_type VARCHAR(16) NOT NULL,
			owner_id VARCHAR(64) NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			file_path VARCHAR(512) NOT NULL,
			file_size BIGINT DEFAULT 0,
			mime_type VARCHAR(128) DEFAULT '',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_model_id ON annotations(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_status ON annotations(status)`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_priority ON annotations(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_annotations_element_id ON annotations(element_id)`,
		`CREATE INDEX IF NOT EXISTS idx_annotation_comments_annotation_id ON annotation_comments(annotation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_annotation_attachments_owner ON annotation_attachments(owner_type, owner_id)`,
	}
	for _, m := range migrations {
		if _, err := r.db.Exec(m); err != nil {
			return fmt.Errorf("annotation migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

func (r *PostgresRepo) CreateAnnotation(a *model.Annotation) error {
	positionStr := fmt.Sprintf(`[%f,%f,%f]`, a.Position[0], a.Position[1], a.Position[2])
	_, err := r.db.Exec(
		`INSERT INTO annotations (id, model_id, type, element_id, position, title, description, priority, status, creator, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`,
		a.ID, a.ModelID, string(a.Type), a.ElementID, positionStr,
		a.Title, a.Description, string(a.Priority), string(a.Status), a.Creator,
	)
	return err
}

func (r *PostgresRepo) GetAnnotation(id string) (*model.Annotation, error) {
	row := r.db.QueryRow(
		`SELECT id, model_id, type, element_id, position, title, description, priority, status, creator, created_at, updated_at
		 FROM annotations WHERE id = $1`, id,
	)
	a, err := r.scanAnnotation(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (r *PostgresRepo) ListAnnotations(q *model.AnnotationListQuery) ([]*model.Annotation, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("model_id = $%d", argIdx))
	args = append(args, q.ModelID)
	argIdx++

	if q.Priority != "" {
		conditions = append(conditions, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(q.Priority))
		argIdx++
	}
	if q.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(q.Status))
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countRow := r.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM annotations WHERE %s", whereClause), args...)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}

	sortBy := "a.created_at"
	sortDir := "DESC"
	switch q.SortBy {
	case "lastReply":
		sortBy = "COALESCE(lc.last_comment_at, a.created_at)"
		sortDir = "DESC"
	case "createdAt":
		sortBy = "a.created_at"
		sortDir = "DESC"
	}

	offset := (q.Page - 1) * q.PageSize

	var query string
	if q.SortBy == "lastReply" {
		query = fmt.Sprintf(
			`SELECT a.id, a.model_id, a.type, a.element_id, a.position, a.title, a.description, a.priority, a.status, a.creator, a.created_at, a.updated_at
			 FROM annotations a
			 LEFT JOIN (SELECT annotation_id, MAX(created_at) as last_comment_at FROM annotation_comments GROUP BY annotation_id) lc ON a.id = lc.annotation_id
			 WHERE %s
			 ORDER BY %s %s
			 LIMIT $%d OFFSET $%d`,
			whereClause, sortBy, sortDir, argIdx, argIdx+1,
		)
	} else {
		query = fmt.Sprintf(
			`SELECT id, model_id, type, element_id, position, title, description, priority, status, creator, created_at, updated_at
			 FROM annotations
			 WHERE %s
			 ORDER BY %s %s
			 LIMIT $%d OFFSET $%d`,
			whereClause, sortBy, sortDir, argIdx, argIdx+1,
		)
	}
	args = append(args, q.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var annotations []*model.Annotation
	for rows.Next() {
		a, err := r.scanAnnotationFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		annotations = append(annotations, a)
	}
	return annotations, total, nil
}

func (r *PostgresRepo) UpdateAnnotation(id string, req *model.UpdateAnnotationRequest) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*req.Priority))
		argIdx++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
	}
	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE annotations SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PostgresRepo) DeleteAnnotation(id string) error {
	_, err := r.db.Exec(`DELETE FROM annotations WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) CreateComment(c *model.Comment) error {
	_, err := r.db.Exec(
		`INSERT INTO annotation_comments (id, annotation_id, content, author, created_at)
		 VALUES ($1, $2, $3, $4, NOW())`,
		c.ID, c.AnnotationID, c.Content, c.Author,
	)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`UPDATE annotations SET updated_at = NOW() WHERE id = $1`, c.AnnotationID)
	return err
}

func (r *PostgresRepo) GetCommentsByAnnotation(annotationID string) ([]*model.Comment, error) {
	rows, err := r.db.Query(
		`SELECT id, annotation_id, content, author, created_at
		 FROM annotation_comments WHERE annotation_id = $1 ORDER BY created_at ASC`,
		annotationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		c := &model.Comment{}
		if err := rows.Scan(&c.ID, &c.AnnotationID, &c.Content, &c.Author, &c.CreatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *PostgresRepo) CreateAttachment(att *model.Attachment) error {
	_, err := r.db.Exec(
		`INSERT INTO annotation_attachments (id, owner_type, owner_id, file_name, file_path, file_size, mime_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		att.ID, att.OwnerType, att.OwnerID, att.FileName, att.FilePath, att.FileSize, att.MimeType,
	)
	return err
}

func (r *PostgresRepo) GetAttachmentsByOwner(ownerType, ownerID string) ([]*model.Attachment, error) {
	rows, err := r.db.Query(
		`SELECT id, owner_type, owner_id, file_name, file_path, file_size, mime_type, created_at
		 FROM annotation_attachments WHERE owner_type = $1 AND owner_id = $2 ORDER BY created_at ASC`,
		ownerType, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*model.Attachment
	for rows.Next() {
		a := &model.Attachment{}
		if err := rows.Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.FileName, &a.FilePath, &a.FileSize, &a.MimeType, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (r *PostgresRepo) GetAttachmentsByOwnerMap(ownerType string, ownerIDs []string) (map[string][]*model.Attachment, error) {
	result := make(map[string][]*model.Attachment)
	if len(ownerIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(ownerIDs))
	args := make([]interface{}, len(ownerIDs)+1)
	args[0] = ownerType
	for i, id := range ownerIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(
		`SELECT id, owner_type, owner_id, file_name, file_path, file_size, mime_type, created_at
		 FROM annotation_attachments WHERE owner_type = $1 AND owner_id IN (%s) ORDER BY created_at ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a := &model.Attachment{}
		if err := rows.Scan(&a.ID, &a.OwnerType, &a.OwnerID, &a.FileName, &a.FilePath, &a.FileSize, &a.MimeType, &a.CreatedAt); err != nil {
			return nil, err
		}
		result[a.OwnerID] = append(result[a.OwnerID], a)
	}
	return result, nil
}

func (r *PostgresRepo) GetAnnotationsByModelSince(modelID string, since time.Time) ([]*model.Annotation, error) {
	rows, err := r.db.Query(
		`SELECT id, model_id, type, element_id, position, title, description, priority, status, creator, created_at, updated_at
		 FROM annotations WHERE model_id = $1 AND updated_at > $2 ORDER BY created_at ASC`,
		modelID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var annotations []*model.Annotation
	for rows.Next() {
		a, err := r.scanAnnotationFromRows(rows)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, a)
	}
	return annotations, nil
}

func (r *PostgresRepo) GetCommentsByAnnotationIDs(annotationIDs []string) (map[string][]*model.Comment, error) {
	result := make(map[string][]*model.Comment)
	if len(annotationIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(annotationIDs))
	args := make([]interface{}, len(annotationIDs))
	for i, id := range annotationIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, annotation_id, content, author, created_at
		 FROM annotation_comments WHERE annotation_id IN (%s) ORDER BY created_at ASC`,
		strings.Join(placeholders, ","),
	)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		c := &model.Comment{}
		if err := rows.Scan(&c.ID, &c.AnnotationID, &c.Content, &c.Author, &c.CreatedAt); err != nil {
			return nil, err
		}
		result[c.AnnotationID] = append(result[c.AnnotationID], c)
	}
	return result, nil
}

func (r *PostgresRepo) scanAnnotation(row *sql.Row) (*model.Annotation, error) {
	a := &model.Annotation{}
	var typeStr, priorityStr, statusStr string
	var posStr string
	err := row.Scan(&a.ID, &a.ModelID, &typeStr, &a.ElementID, &posStr, &a.Title, &a.Description, &priorityStr, &statusStr, &a.Creator, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	var pos [3]float64
	parsePosition(posStr, &pos)
	a.Position = pos
	a.Type = model.AnnotationType(typeStr)
	a.Priority = model.AnnotationPriority(priorityStr)
	a.Status = model.AnnotationStatus(statusStr)
	return a, nil
}

func (r *PostgresRepo) scanAnnotationFromRows(rows *sql.Rows) (*model.Annotation, error) {
	a := &model.Annotation{}
	var typeStr, priorityStr, statusStr string
	var posStr string
	err := rows.Scan(&a.ID, &a.ModelID, &typeStr, &a.ElementID, &posStr, &a.Title, &a.Description, &priorityStr, &statusStr, &a.Creator, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	var pos [3]float64
	parsePosition(posStr, &pos)
	a.Position = pos
	a.Type = model.AnnotationType(typeStr)
	a.Priority = model.AnnotationPriority(priorityStr)
	a.Status = model.AnnotationStatus(statusStr)
	return a, nil
}

func parsePosition(s string, target *[3]float64) {
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")
	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(strings.TrimSpace(parts[i]), "%f", &target[i])
	}
}
