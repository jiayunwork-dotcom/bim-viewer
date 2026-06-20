package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"bim-viewer/internal/model"
	"bim-viewer/internal/service"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type AnnotationHandler struct {
	annotationSvc *service.AnnotationService
	wsHub         *service.WSHub
	uploader      websocket.Upgrader
}

func NewAnnotationHandler(annotationSvc *service.AnnotationService, wsHub *service.WSHub) *AnnotationHandler {
	return &AnnotationHandler{
		annotationSvc: annotationSvc,
		wsHub:         wsHub,
		uploader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *AnnotationHandler) CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	req := &model.CreateAnnotationRequest{
		ModelID:     r.FormValue("modelId"),
		Type:        model.AnnotationType(r.FormValue("type")),
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		Priority:    model.AnnotationPriority(r.FormValue("priority")),
		Creator:     r.FormValue("creator"),
	}

	if issueID := r.FormValue("issueId"); issueID != "" {
		req.IssueID = &issueID
	}

	if elementID := r.FormValue("elementId"); elementID != "" {
		req.ElementID = &elementID
	}

	if posStr := r.FormValue("position"); posStr != "" {
		var pos [3]float64
		if err := json.Unmarshal([]byte(posStr), &pos); err == nil {
			req.Position = pos
		}
	}

	if req.ModelID == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "modelId and title are required")
		return
	}
	if req.Priority == "" {
		req.Priority = model.PriorityNormal
	}
	if req.Type == "" {
		req.Type = model.AnnotationTypeElement
	}
	if req.Creator == "" {
		req.Creator = "anonymous"
	}

	annotation, err := h.annotationSvc.CreateAnnotation(req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "issueId is required" || errMsg == "issue not found" || 
		   errMsg == "issue does not belong to this model" || errMsg == "cannot add annotation to archived issue" {
			writeError(w, http.StatusBadRequest, errMsg)
		} else {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create annotation: %v", err))
		}
		return
	}

	files := r.MultipartForm.File["attachments"]
	if len(files) > 3 {
		files = files[:3]
	}

	for _, fh := range files {
		if fh.Size > 5*1024*1024 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("File %s exceeds 5MB limit", fh.Filename))
			return
		}

		f, err := fh.Open()
		if err != nil {
			continue
		}
		fileData, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		mimeType := fh.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		h.annotationSvc.SaveAttachment("annotation", annotation.ID, fh.Filename, fileData, fh.Size, mimeType)
	}

	fullAnnotation, _ := h.annotationSvc.GetAnnotation(annotation.ID)
	if fullAnnotation != nil {
		annotation = fullAnnotation
	}

	writeJSON(w, http.StatusCreated, annotation)
}

func (h *AnnotationHandler) GetAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	annotation, err := h.annotationSvc.GetAnnotation(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get annotation")
		return
	}
	if annotation == nil {
		writeError(w, http.StatusNotFound, "Annotation not found")
		return
	}
	writeJSON(w, http.StatusOK, annotation)
}

func (h *AnnotationHandler) ListAnnotations(w http.ResponseWriter, r *http.Request) {
	q := &model.AnnotationListQuery{
		ModelID: r.URL.Query().Get("modelId"),
		Page:    1,
		PageSize: 20,
	}

	if issueID := r.URL.Query().Get("issueId"); issueID != "" {
		q.IssueID = issueID
	}
	if p := r.URL.Query().Get("priority"); p != "" {
		q.Priority = model.AnnotationPriority(p)
	}
	if s := r.URL.Query().Get("status"); s != "" {
		q.Status = model.AnnotationStatus(s)
	}
	if s := r.URL.Query().Get("sortBy"); s != "" {
		q.SortBy = s
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		q.Page = p
	}
	if ps, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && ps > 0 && ps <= 100 {
		q.PageSize = ps
	}

	if q.ModelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}

	resp, err := h.annotationSvc.ListAnnotations(q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list annotations: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AnnotationHandler) UpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	currentUser := r.Header.Get("X-Current-User")

	var req model.UpdateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	annotation, err := h.annotationSvc.UpdateAnnotation(id, &req, currentUser)
	if err != nil {
		if err.Error() == "permission denied: only creator can edit title and description" {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update annotation: %v", err))
		return
	}
	if annotation == nil {
		writeError(w, http.StatusNotFound, "Annotation not found")
		return
	}
	writeJSON(w, http.StatusOK, annotation)
}

func (h *AnnotationHandler) DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	currentUser := r.Header.Get("X-Current-User")

	if err := h.annotationSvc.DeleteAnnotation(id, currentUser); err != nil {
		if err.Error() == "permission denied: only creator can delete annotation" {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete annotation: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AnnotationHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	commentID := vars["commentId"]
	currentUser := r.Header.Get("X-Current-User")

	if err := h.annotationSvc.DeleteComment(commentID, currentUser); err != nil {
		if err.Error() == "permission denied: only author can delete comment" {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete comment: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AnnotationHandler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	var req model.CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ModelID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "modelId and name are required")
		return
	}
	if req.Creator == "" {
		req.Creator = "anonymous"
	}

	issue, err := h.annotationSvc.CreateIssue(&req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create issue: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, issue)
}

func (h *AnnotationHandler) GetIssue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	issue, err := h.annotationSvc.GetIssue(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get issue")
		return
	}
	if issue == nil {
		writeError(w, http.StatusNotFound, "Issue not found")
		return
	}
	writeJSON(w, http.StatusOK, issue)
}

func (h *AnnotationHandler) ListIssues(w http.ResponseWriter, r *http.Request) {
	q := &model.IssueListQuery{
		ModelID: r.URL.Query().Get("modelId"),
	}

	if status := r.URL.Query().Get("status"); status != "" {
		q.Status = model.IssueStatus(status)
	}

	if q.ModelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}

	issues, err := h.annotationSvc.ListIssues(q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list issues: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

func (h *AnnotationHandler) UpdateIssue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	currentUser := r.Header.Get("X-Current-User")

	var req model.UpdateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	issue, err := h.annotationSvc.UpdateIssue(id, &req, currentUser)
	if err != nil {
		if err.Error() == "permission denied: only creator can edit issue" {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update issue: %v", err))
		return
	}
	if issue == nil {
		writeError(w, http.StatusNotFound, "Issue not found")
		return
	}
	writeJSON(w, http.StatusOK, issue)
}

func (h *AnnotationHandler) ArchiveIssue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	currentUser := r.Header.Get("X-Current-User")

	issue, err := h.annotationSvc.ArchiveIssue(id, currentUser)
	if err != nil {
		if err.Error() == "permission denied: only creator can edit issue" {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to archive issue: %v", err))
		return
	}
	if issue == nil {
		writeError(w, http.StatusNotFound, "Issue not found")
		return
	}
	writeJSON(w, http.StatusOK, issue)
}

func (h *AnnotationHandler) GetIssuesDueSoon(w http.ResponseWriter, r *http.Request) {
	modelID := r.URL.Query().Get("modelId")
	if modelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}

	withinHours := 24
	if hStr := r.URL.Query().Get("withinHours"); hStr != "" {
		if h, err := strconv.Atoi(hStr); err == nil && h > 0 {
			withinHours = h
		}
	}

	issues, err := h.annotationSvc.GetIssuesDueSoon(modelID, withinHours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get due soon issues: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, issues)
}

func (h *AnnotationHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	annotationID := vars["id"]

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	req := &model.CreateCommentRequest{
		Content: r.FormValue("content"),
		Author:  r.FormValue("author"),
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Author == "" {
		req.Author = "anonymous"
	}

	comment, err := h.annotationSvc.AddComment(annotationID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to add comment: %v", err))
		return
	}

	files := r.MultipartForm.File["attachment"]
	if len(files) > 1 {
		files = files[:1]
	}

	for _, fh := range files {
		if fh.Size > 5*1024*1024 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("File %s exceeds 5MB limit", fh.Filename))
			return
		}

		f, err := fh.Open()
		if err != nil {
			continue
		}
		fileData, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		mimeType := fh.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		att, err := h.annotationSvc.SaveAttachment("comment", comment.ID, fh.Filename, fileData, fh.Size, mimeType)
		if err == nil {
			comment.Attachment = att
		}
	}

	writeJSON(w, http.StatusCreated, comment)
}

func (h *AnnotationHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	annotationID := vars["id"]

	comments, err := h.annotationSvc.GetComments(annotationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get comments")
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *AnnotationHandler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	fileName := mux.Vars(r)["filename"]
	uploadDir := h.annotationSvc.GetUploadDir()
	filePath := filepath.Join(uploadDir, filepath.Base(fileName))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "File not found")
		return
	}

	http.ServeFile(w, r, filePath)
}

func (h *AnnotationHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	modelID := r.URL.Query().Get("modelId")
	if modelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}

	conn, err := h.uploader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &service.WSClient{
		Hub:     h.wsHub,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		ModelID: modelID,
	}

	h.wsHub.Register <- client

	go client.WritePump()

	repo := h.annotationSvc.GetRepo()
	go client.ReadPump(repo)

	syncMsg, _ := json.Marshal(model.WSMessage{
		Type:      "connected",
		ModelID:   modelID,
		Payload:   map[string]string{"status": "connected", "serverTime": time.Now().Format(time.RFC3339)},
		Timestamp: time.Now(),
	})
	client.Send <- syncMsg
}

func (h *AnnotationHandler) GetAnnotationsSince(w http.ResponseWriter, r *http.Request) {
	modelID := r.URL.Query().Get("modelId")
	sinceStr := r.URL.Query().Get("since")

	if modelID == "" || sinceStr == "" {
		writeError(w, http.StatusBadRequest, "modelId and since are required")
		return
	}

	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid since timestamp format, use RFC3339")
		return
	}

	annotations, err := h.annotationSvc.GetAnnotationsSince(modelID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get annotations")
		return
	}
	writeJSON(w, http.StatusOK, annotations)
}

var _ = os.DevNull
var _ = filepath.Base
var _ = strconv.Atoi
