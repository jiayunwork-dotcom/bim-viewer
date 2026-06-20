package service

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"bim-viewer/internal/model"
	"bim-viewer/internal/repository"
)

type AnnotationService struct {
	repo        *repository.PostgresRepo
	wsHub       *WSHub
	uploadDir   string
}

func NewAnnotationService(repo *repository.PostgresRepo, wsHub *WSHub) *AnnotationService {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/tmp/bim-uploads"
	}
	os.MkdirAll(uploadDir, 0755)

	return &AnnotationService{
		repo:      repo,
		wsHub:     wsHub,
		uploadDir: uploadDir,
	}
}

func (s *AnnotationService) CreateAnnotation(req *model.CreateAnnotationRequest) (*model.Annotation, error) {
	if req.IssueID == nil || *req.IssueID == "" {
		return nil, fmt.Errorf("issueId is required")
	}

	issue, err := s.repo.GetIssue(*req.IssueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}
	if issue == nil {
		return nil, fmt.Errorf("issue not found")
	}
	if issue.ModelID != req.ModelID {
		return nil, fmt.Errorf("issue does not belong to this model")
	}
	if issue.Status != model.IssueStatusActive {
		return nil, fmt.Errorf("cannot add annotation to archived issue")
	}

	a := &model.Annotation{
		ID:          generateAnnotationUUID(),
		ModelID:     req.ModelID,
		IssueID:     req.IssueID,
		Type:        req.Type,
		ElementID:   req.ElementID,
		Position:    req.Position,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      model.AnnotationStatusOpen,
		Creator:     req.Creator,
	}

	if err := s.repo.CreateAnnotation(a); err != nil {
		return nil, fmt.Errorf("failed to create annotation: %w", err)
	}

	fullAnnotation, _ := s.GetAnnotation(a.ID)
	if fullAnnotation != nil {
		a = fullAnnotation
	}

	if s.wsHub != nil {
		s.wsHub.BroadcastToModel(req.ModelID, model.WSMessage{
			Type:      "annotation_created",
			ModelID:   req.ModelID,
			Payload:   a,
			Timestamp: time.Now(),
		})
	}

	return a, nil
}

func (s *AnnotationService) CreateIssue(req *model.CreateIssueRequest) (*model.Issue, error) {
	issue := &model.Issue{
		ID:          generateIssueUUID(),
		ModelID:     req.ModelID,
		Name:        req.Name,
		Description: req.Description,
		Owner:       req.Owner,
		Status:      model.IssueStatusActive,
		Creator:     req.Creator,
	}

	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err == nil {
			issue.DueDate = &t
		}
	}

	if err := s.repo.CreateIssue(issue); err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	if s.wsHub != nil {
		s.wsHub.BroadcastToModel(req.ModelID, model.WSMessage{
			Type:      "issue_created",
			ModelID:   req.ModelID,
			Payload:   issue,
			Timestamp: time.Now(),
		})
	}

	return issue, nil
}

func (s *AnnotationService) GetIssue(id string) (*model.Issue, error) {
	return s.repo.GetIssue(id)
}

func (s *AnnotationService) ListIssues(q *model.IssueListQuery) ([]*model.Issue, error) {
	return s.repo.ListIssues(q)
}

func (s *AnnotationService) UpdateIssue(id string, req *model.UpdateIssueRequest, currentUser string) (*model.Issue, error) {
	issue, err := s.repo.GetIssue(id)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, nil
	}

	if issue.Creator != currentUser {
		return nil, fmt.Errorf("permission denied: only creator can edit issue")
	}

	req.CurrentUser = currentUser
	if err := s.repo.UpdateIssue(id, req); err != nil {
		return nil, err
	}

	updatedIssue, err := s.repo.GetIssue(id)
	if err != nil {
		return nil, err
	}

	if s.wsHub != nil && updatedIssue != nil {
		s.wsHub.BroadcastToModel(updatedIssue.ModelID, model.WSMessage{
			Type:      "issue_updated",
			ModelID:   updatedIssue.ModelID,
			Payload:   updatedIssue,
			Timestamp: time.Now(),
		})
	}

	return updatedIssue, nil
}

func (s *AnnotationService) ArchiveIssue(id string, currentUser string) (*model.Issue, error) {
	archivedStatus := string(model.IssueStatusArchived)
	req := &model.UpdateIssueRequest{
		Status: &archivedStatus,
	}
	return s.UpdateIssue(id, req, currentUser)
}

func (s *AnnotationService) GetIssuesDueSoon(modelID string, withinHours int) ([]*model.Issue, error) {
	return s.repo.GetIssuesDueSoon(modelID, withinHours)
}

func (s *AnnotationService) GetAnnotation(id string) (*model.Annotation, error) {
	a, err := s.repo.GetAnnotation(id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, nil
	}

	if a.IssueID != nil {
		issue, err := s.repo.GetIssue(*a.IssueID)
		if err == nil && issue != nil {
			a.Issue = issue
		}
	}

	attachments, err := s.repo.GetAttachmentsByOwner("annotation", id)
	if err != nil {
		return nil, err
	}
	a.Attachments = make([]model.Attachment, len(attachments))
	for i, att := range attachments {
		a.Attachments[i] = *att
	}

	comments, err := s.repo.GetCommentsByAnnotation(id)
	if err != nil {
		return nil, err
	}

	commentIDs := make([]string, len(comments))
	for i, c := range comments {
		commentIDs[i] = c.ID
	}

	commentAttMap, err := s.repo.GetAttachmentsByOwnerMap("comment", commentIDs)
	if err != nil {
		return nil, err
	}

	a.Comments = make([]model.Comment, len(comments))
	for i, c := range comments {
		a.Comments[i] = *c
		if atts, ok := commentAttMap[c.ID]; ok && len(atts) > 0 {
			a.Comments[i].Attachment = atts[0]
		}
	}

	return a, nil
}

func (s *AnnotationService) ListAnnotations(q *model.AnnotationListQuery) (*model.AnnotationListResponse, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}

	annotations, total, err := s.repo.ListAnnotations(q)
	if err != nil {
		return nil, err
	}

	if annotations == nil {
		annotations = []*model.Annotation{}
	}

	ids := make([]string, len(annotations))
	issueIDs := make([]string, 0)
	issueIDSet := make(map[string]bool)
	for i, a := range annotations {
		ids[i] = a.ID
		if a.IssueID != nil && !issueIDSet[*a.IssueID] {
			issueIDs = append(issueIDs, *a.IssueID)
			issueIDSet[*a.IssueID] = true
		}
	}

	attMap, err := s.repo.GetAttachmentsByOwnerMap("annotation", ids)
	if err != nil {
		return nil, err
	}

	commentMap, err := s.repo.GetCommentsByAnnotationIDs(ids)
	if err != nil {
		return nil, err
	}

	issueMap := make(map[string]*model.Issue)
	if len(issueIDs) > 0 {
		issueQuery := &model.IssueListQuery{ModelID: q.ModelID}
		issues, _ := s.repo.ListIssues(issueQuery)
		for _, issue := range issues {
			issueMap[issue.ID] = issue
		}
	}

	items := make([]model.Annotation, len(annotations))
	for i, a := range annotations {
		items[i] = *a
		if a.IssueID != nil {
			if issue, ok := issueMap[*a.IssueID]; ok {
				items[i].Issue = issue
			}
		}
		if atts, ok := attMap[a.ID]; ok {
			items[i].Attachments = make([]model.Attachment, len(atts))
			for j, att := range atts {
				items[i].Attachments[j] = *att
			}
		}
		if comments, ok := commentMap[a.ID]; ok {
			items[i].Comments = make([]model.Comment, len(comments))
			for j, c := range comments {
				items[i].Comments[j] = *c
			}
		}
	}

	totalPages := total / q.PageSize
	if total%q.PageSize > 0 {
		totalPages++
	}

	return &model.AnnotationListResponse{
		Items:      items,
		Total:      total,
		Page:       q.Page,
		PageSize:   q.PageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *AnnotationService) UpdateAnnotation(id string, req *model.UpdateAnnotationRequest, currentUser string) (*model.Annotation, error) {
	existing, err := s.repo.GetAnnotation(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	hasTitleOrDesc := req.Title != nil || req.Description != nil
	if hasTitleOrDesc && existing.Creator != currentUser {
		return nil, fmt.Errorf("permission denied: only creator can edit title and description")
	}

	req.CurrentUser = currentUser
	if err := s.repo.UpdateAnnotation(id, req); err != nil {
		return nil, err
	}

	a, err := s.GetAnnotation(id)
	if err != nil {
		return nil, err
	}

	if s.wsHub != nil && a != nil {
		s.wsHub.BroadcastToModel(a.ModelID, model.WSMessage{
			Type:      "annotation_updated",
			ModelID:   a.ModelID,
			Payload:   a,
			Timestamp: time.Now(),
		})
	}

	return a, nil
}

func (s *AnnotationService) DeleteAnnotation(id string, currentUser string) error {
	a, err := s.repo.GetAnnotation(id)
	if err != nil {
		return err
	}
	if a == nil {
		return nil
	}

	if a.Creator != currentUser {
		return fmt.Errorf("permission denied: only creator can delete annotation")
	}

	modelID := a.ModelID
	if err := s.repo.DeleteAnnotation(id); err != nil {
		return err
	}

	if s.wsHub != nil {
		s.wsHub.BroadcastToModel(modelID, model.WSMessage{
			Type:      "annotation_deleted",
			ModelID:   modelID,
			Payload:   map[string]string{"id": id},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (s *AnnotationService) DeleteComment(commentID string, currentUser string) error {
	comment, err := s.repo.GetComment(commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return nil
	}

	if comment.Author != currentUser {
		return fmt.Errorf("permission denied: only author can delete comment")
	}

	annotation, _ := s.repo.GetAnnotation(comment.AnnotationID)
	if err := s.repo.DeleteComment(commentID); err != nil {
		return err
	}

	if s.wsHub != nil && annotation != nil {
		s.wsHub.BroadcastToModel(annotation.ModelID, model.WSMessage{
			Type:      "comment_deleted",
			ModelID:   annotation.ModelID,
			Payload:   map[string]string{"id": commentID, "annotationId": comment.AnnotationID},
			Timestamp: time.Now(),
		})
	}

	return nil
}

func (s *AnnotationService) AddComment(annotationID string, req *model.CreateCommentRequest) (*model.Comment, error) {
	c := &model.Comment{
		ID:           generateAnnotationUUID(),
		AnnotationID: annotationID,
		Content:      req.Content,
		Author:       req.Author,
	}

	if err := s.repo.CreateComment(c); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	a, _ := s.repo.GetAnnotation(annotationID)
	if s.wsHub != nil && a != nil {
		s.wsHub.BroadcastToModel(a.ModelID, model.WSMessage{
			Type:      "comment_added",
			ModelID:   a.ModelID,
			Payload:   c,
			Timestamp: time.Now(),
		})
	}

	return c, nil
}

func (s *AnnotationService) GetComments(annotationID string) ([]*model.Comment, error) {
	comments, err := s.repo.GetCommentsByAnnotation(annotationID)
	if err != nil {
		return nil, err
	}
	if comments == nil {
		comments = []*model.Comment{}
	}
	return comments, nil
}

func (s *AnnotationService) SaveAttachment(ownerType, ownerID, fileName string, fileData []byte, fileSize int64, mimeType string) (*model.Attachment, error) {
	ext := filepath.Ext(fileName)
	storedName := fmt.Sprintf("%s_%s%s", ownerID, generateAnnotationUUID(), ext)
	filePath := filepath.Join(s.uploadDir, storedName)

	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	att := &model.Attachment{
		ID:        generateAnnotationUUID(),
		OwnerType: ownerType,
		OwnerID:   ownerID,
		FileName:  fileName,
		FilePath:  storedName,
		FileSize:  fileSize,
		MimeType:  mimeType,
	}

	if err := s.repo.CreateAttachment(att); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create attachment record: %w", err)
	}

	return att, nil
}

func (s *AnnotationService) GetUploadDir() string {
	return s.uploadDir
}

func (s *AnnotationService) GetRepo() *repository.PostgresRepo {
	return s.repo
}

func (s *AnnotationService) GetAnnotationsSince(modelID string, since time.Time) ([]*model.Annotation, error) {
	annotations, err := s.repo.GetAnnotationsByModelSince(modelID, since)
	if err != nil {
		return nil, err
	}
	if annotations == nil {
		annotations = []*model.Annotation{}
	}
	return annotations, nil
}

func generateAnnotationUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("ann-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func generateIssueUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("issue-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("iss-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
