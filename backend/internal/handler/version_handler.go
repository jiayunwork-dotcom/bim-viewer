package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"bim-viewer/internal/model"
	"bim-viewer/internal/service"

	"github.com/gorilla/mux"
)

type VersionHandler struct {
	modelSvc *service.ModelService
}

func NewVersionHandler(modelSvc *service.ModelService) *VersionHandler {
	return &VersionHandler{modelSvc: modelSvc}
}

func (h *VersionHandler) CreateVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	var req model.CreateVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	version, err := h.modelSvc.CreateVersion(modelID, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, version)
}

func (h *VersionHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	versions, err := h.modelSvc.ListVersions(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list versions")
		return
	}
	if versions == nil {
		versions = []*model.ModelVersion{}
	}
	writeJSON(w, http.StatusOK, versions)
}

func (h *VersionHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]

	version, err := h.modelSvc.GetVersion(versionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get version")
		return
	}
	if version == nil {
		writeError(w, http.StatusNotFound, "Version not found")
		return
	}
	writeJSON(w, http.StatusOK, version)
}

func (h *VersionHandler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]

	if err := h.modelSvc.DeleteVersion(versionID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete version")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *VersionHandler) CompareVersions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	var req model.CompareVersionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BaseVersionID == "" || req.CompareVersionID == "" {
		writeError(w, http.StatusBadRequest, "Both baseVersionId and compareVersionId are required")
		return
	}

	result, err := h.modelSvc.CompareVersions(req.BaseVersionID, req.CompareVersionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = modelID
	writeJSON(w, http.StatusOK, result)
}

func (h *VersionHandler) GetVersionElement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	versionID := vars["versionId"]
	elementID := vars["elementId"]

	element, err := h.modelSvc.GetVersionElement(versionID, elementID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get element")
		return
	}
	if element == nil {
		writeError(w, http.StatusNotFound, "Element not found in this version")
		return
	}
	writeJSON(w, http.StatusOK, element)
}

func (h *VersionHandler) CreateVersionAnnotation(w http.ResponseWriter, r *http.Request) {
	currentUser := r.Header.Get("X-Current-User")

	var req model.CreateVersionAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BaseVersionID == "" || req.CompareVersionID == "" || req.ElementID == "" {
		writeError(w, http.StatusBadRequest, "baseVersionId, compareVersionId and elementId are required")
		return
	}

	trimmedContent := strings.TrimSpace(req.Content)
	if trimmedContent == "" {
		writeError(w, http.StatusBadRequest, "批注内容不能为空")
		return
	}
	if len(trimmedContent) > 500 {
		writeError(w, http.StatusBadRequest, "批注内容不能超过500字符")
		return
	}

	if currentUser != "" {
		req.Author = currentUser
	}

	annotation, err := h.modelSvc.CreateVersionAnnotation(&req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "批注内容不能为空" || errMsg == "批注内容不能超过500字符" ||
			errMsg == "base version not found" || errMsg == "compare version not found" ||
			errMsg == "versions must belong to the same model" {
			writeError(w, http.StatusBadRequest, errMsg)
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to create annotation")
		}
		return
	}

	writeJSON(w, http.StatusCreated, annotation)
}

func (h *VersionHandler) ListVersionAnnotations(w http.ResponseWriter, r *http.Request) {
	baseVersionID := r.URL.Query().Get("baseVersionId")
	compareVersionID := r.URL.Query().Get("compareVersionId")

	if baseVersionID == "" || compareVersionID == "" {
		writeError(w, http.StatusBadRequest, "baseVersionId and compareVersionId are required")
		return
	}

	annotations, err := h.modelSvc.ListVersionAnnotations(baseVersionID, compareVersionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list annotations")
		return
	}
	if annotations == nil {
		annotations = []*model.VersionAnnotation{}
	}
	writeJSON(w, http.StatusOK, annotations)
}

func (h *VersionHandler) DeleteVersionAnnotation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	currentUser := r.Header.Get("X-Current-User")

	if err := h.modelSvc.DeleteVersionAnnotation(id, currentUser); err != nil {
		errMsg := err.Error()
		if errMsg == "annotation not found" {
			writeError(w, http.StatusNotFound, errMsg)
		} else if errMsg == "permission denied: only author can delete this annotation" {
			writeError(w, http.StatusForbidden, errMsg)
		} else {
			writeError(w, http.StatusInternalServerError, "Failed to delete annotation")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *VersionHandler) ExportCompareReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["modelId"]

	var req model.CompareVersionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.BaseVersionID == "" || req.CompareVersionID == "" {
		writeError(w, http.StatusBadRequest, "Both baseVersionId and compareVersionId are required")
		return
	}

	_ = modelID

	report, err := h.modelSvc.GenerateVersionCompareReport(req.BaseVersionID, req.CompareVersionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, report)
}
