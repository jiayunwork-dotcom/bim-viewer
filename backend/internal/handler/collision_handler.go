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

func (h *CollisionHandler) GetCollisionStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	stats, err := h.modelSvc.GetCollisionStats(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *CollisionHandler) GetCollisionStatsByModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	stats, err := h.modelSvc.GetCollisionStatsByModel(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *CollisionHandler) GetCollisionResultsByModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	results, err := h.modelSvc.GetCollisionResultsByModel(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision results")
		return
	}
	if results == nil {
		results = []*model.CollisionResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *CollisionHandler) GetCollisionHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resultID := vars["resultId"]

	history, err := h.modelSvc.GetCollisionHistory(resultID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision history")
		return
	}
	if history == nil {
		history = []*model.CollisionResultHistory{}
	}
	writeJSON(w, http.StatusOK, history)
}

func (h *CollisionHandler) UpdateCollisionStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resultID := vars["resultId"]

	var req struct {
		NewStatus model.CollisionStatus `json:"newStatus"`
		Remark    string                `json:"remark"`
		Operator  string                `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Remark == "" {
		writeError(w, http.StatusBadRequest, "Remark is required")
		return
	}

	if err := h.modelSvc.UpdateCollisionResultStatus(resultID, req.NewStatus, req.Remark, req.Operator); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update status: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Status updated successfully",
	})
}

func (h *CollisionHandler) BatchUpdateCollisionStatus(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.ResultIDs) == 0 {
		writeError(w, http.StatusBadRequest, "resultIds is required and cannot be empty")
		return
	}

	if req.Remark == "" {
		writeError(w, http.StatusBadRequest, "Remark is required")
		return
	}

	updated, err := h.modelSvc.BatchUpdateCollisionStatus(req.ResultIDs, req.NewStatus, req.Remark, req.Operator)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to batch update status: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"updated": updated,
		"message": fmt.Sprintf("Successfully updated %d records", updated),
	})
}

func (h *CollisionHandler) GetCollisionTasksByModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	tasks, err := h.modelSvc.GetCollisionTasksByModel(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get collision tasks")
		return
	}
	if tasks == nil {
		tasks = []*model.CollisionTask{}
	}
	writeJSON(w, http.StatusOK, tasks)
}
