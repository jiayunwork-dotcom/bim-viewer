package model

import "time"

type AnnotationType string

const (
	AnnotationTypeElement AnnotationType = "element"
	AnnotationTypeSpace   AnnotationType = "space"
)

type AnnotationPriority string

const (
	PriorityUrgent AnnotationPriority = "urgent"
	PriorityNormal AnnotationPriority = "normal"
	PriorityLow    AnnotationPriority = "low"
)

type AnnotationStatus string

const (
	AnnotationStatusOpen       AnnotationStatus = "open"
	AnnotationStatusInProgress AnnotationStatus = "in_progress"
	AnnotationStatusClosed     AnnotationStatus = "closed"
)

type Annotation struct {
	ID          string            `json:"id"`
	ModelID     string            `json:"modelId"`
	Type        AnnotationType    `json:"type"`
	ElementID   *string           `json:"elementId,omitempty"`
	Position    [3]float64        `json:"position"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Priority    AnnotationPriority `json:"priority"`
	Status      AnnotationStatus  `json:"status"`
	Creator     string            `json:"creator"`
	Attachments []Attachment      `json:"attachments,omitempty"`
	Comments    []Comment         `json:"comments,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type Comment struct {
	ID           string      `json:"id"`
	AnnotationID string      `json:"annotationId"`
	Content      string      `json:"content"`
	Author       string      `json:"author"`
	Attachment   *Attachment `json:"attachment,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
}

type Attachment struct {
	ID         string    `json:"id"`
	OwnerType  string    `json:"ownerType"`
	OwnerID    string    `json:"ownerId"`
	FileName   string    `json:"fileName"`
	FilePath   string    `json:"filePath"`
	FileSize   int64     `json:"fileSize"`
	MimeType   string    `json:"mimeType"`
	CreatedAt  time.Time `json:"createdAt"`
}

type CreateAnnotationRequest struct {
	ModelID     string             `json:"modelId"`
	Type        AnnotationType     `json:"type"`
	ElementID   *string            `json:"elementId,omitempty"`
	Position    [3]float64         `json:"position"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Priority    AnnotationPriority `json:"priority"`
	Creator     string             `json:"creator"`
}

type UpdateAnnotationRequest struct {
	Priority *AnnotationPriority `json:"priority,omitempty"`
	Status   *AnnotationStatus   `json:"status,omitempty"`
	Title    *string             `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
	Author  string `json:"author"`
}

type AnnotationListQuery struct {
	ModelID  string             `json:"modelId"`
	Priority AnnotationPriority `json:"priority,omitempty"`
	Status   AnnotationStatus   `json:"status,omitempty"`
	SortBy   string             `json:"sortBy,omitempty"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type AnnotationListResponse struct {
	Items      []Annotation `json:"items"`
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	TotalPages int          `json:"totalPages"`
}

type WSMessage struct {
	Type      string      `json:"type"`
	ModelID   string      `json:"modelId"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}
