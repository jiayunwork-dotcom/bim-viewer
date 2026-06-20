package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"bim-viewer/internal/model"
	"bim-viewer/internal/service"

	"github.com/gorilla/mux"
)

type ConstructionHandler struct {
	constructionSvc *service.ConstructionService
}

func NewConstructionHandler(constructionSvc *service.ConstructionService) *ConstructionHandler {
	return &ConstructionHandler{constructionSvc: constructionSvc}
}

func (h *ConstructionHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req model.CreateConstructionPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	plan, err := h.constructionSvc.CreatePlan(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, plan)
}

func (h *ConstructionHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	plan, err := h.constructionSvc.GetPlan(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get construction plan")
		return
	}
	if plan == nil {
		writeError(w, http.StatusNotFound, "Construction plan not found")
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *ConstructionHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	modelID := r.URL.Query().Get("modelId")
	if modelID == "" {
		writeError(w, http.StatusBadRequest, "modelId is required")
		return
	}

	plans, err := h.constructionSvc.ListPlansByModel(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list construction plans: %v", err))
		return
	}
	if plans == nil {
		plans = []*model.ConstructionPlan{}
	}
	writeJSON(w, http.StatusOK, plans)
}

func (h *ConstructionHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req model.UpdateConstructionPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	plan, err := h.constructionSvc.UpdatePlan(id, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if plan == nil {
		writeError(w, http.StatusNotFound, "Construction plan not found")
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (h *ConstructionHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.constructionSvc.DeletePlan(id); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete construction plan: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ConstructionHandler) CreatePhase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planID := vars["planId"]

	var req model.CreateConstructionPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	phase, err := h.constructionSvc.CreatePhase(planID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, phase)
}

func (h *ConstructionHandler) GetPhase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["phaseId"]

	phase, err := h.constructionSvc.GetPhasesByPlan(vars["planId"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get phase")
		return
	}
	var found *model.ConstructionPhase
	for i := range phase {
		if phase[i].ID == id {
			found = &phase[i]
			break
		}
	}
	if found == nil {
		writeError(w, http.StatusNotFound, "Phase not found")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (h *ConstructionHandler) ListPhases(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planID := vars["planId"]

	phases, err := h.constructionSvc.GetPhasesByPlan(planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list phases: %v", err))
		return
	}
	if phases == nil {
		phases = []model.ConstructionPhase{}
	}
	writeJSON(w, http.StatusOK, phases)
}

func (h *ConstructionHandler) UpdatePhase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	phaseID := vars["phaseId"]

	var req model.UpdateConstructionPhaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	phase, err := h.constructionSvc.UpdatePhase(phaseID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if phase == nil {
		writeError(w, http.StatusNotFound, "Phase not found")
		return
	}
	writeJSON(w, http.StatusOK, phase)
}

func (h *ConstructionHandler) DeletePhase(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	phaseID := vars["phaseId"]

	if err := h.constructionSvc.DeletePhase(phaseID); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete phase: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
