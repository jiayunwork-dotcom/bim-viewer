package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bim-viewer/internal/model"
	"bim-viewer/internal/service"

	"github.com/gorilla/mux"
)

type CollisionHandler struct {
	collisionSvc *service.CollisionService
	modelSvc     *service.ModelService
}

func NewCollisionHandler(collisionSvc *service.CollisionService, modelSvc *service.ModelService) *CollisionHandler {
	return &CollisionHandler{
		collisionSvc: collisionSvc,
		modelSvc:     modelSvc,
	}
}

func (h *CollisionHandler) DetectCollisions(w http.ResponseWriter, r *http.Request) {
	var req service.CollisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ModelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}
	if len(req.GroupA) == 0 || len(req.GroupB) == 0 {
		writeError(w, http.StatusBadRequest, "Both groupA and groupB must have at least one element")
		return
	}
	if req.Threshold <= 0 {
		req.Threshold = 50.0
	}

	taskID, results, err := h.modelSvc.RunCollisionDetection(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Collision detection failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"taskId":  taskID,
		"status":  "completed",
		"results": results,
		"count":   len(results),
	})
}

func (h *CollisionHandler) GetCollisionResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	results, err := h.modelSvc.GetCollisionResults(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision results")
		return
	}
	if results == nil {
		results = []*model.CollisionResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *CollisionHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=collision_report_%s.csv", taskID))

	if err := h.modelSvc.ExportCollisionCSV(taskID, w); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to export CSV")
		return
	}
}
