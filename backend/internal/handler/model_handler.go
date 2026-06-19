package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"bim-viewer/internal/model"
	"bim-viewer/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type ModelHandler struct {
	modelSvc *service.ModelService
}

func NewModelHandler(modelSvc *service.ModelService) *ModelHandler {
	return &ModelHandler{modelSvc: modelSvc}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *ModelHandler) UploadModel(w http.ResponseWriter, r *http.Request) {
	maxSize := int64(500 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		writeError(w, http.StatusBadRequest, "File too large or invalid form data")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".ifc" {
		writeError(w, http.StatusBadRequest, "Only IFC files are supported")
		return
	}

	model, err := h.modelSvc.UploadModel(header.Filename, header.Size, file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to process model: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, model)
}

func (h *ModelHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.modelSvc.ListModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list models")
		return
	}
	if models == nil {
		models = []*model.Model{}
	}
	writeJSON(w, http.StatusOK, models)
}

func (h *ModelHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	m, err := h.modelSvc.GetModel(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get model")
		return
	}
	if m == nil {
		writeError(w, http.StatusNotFound, "Model not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *ModelHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.modelSvc.DeleteModel(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete model")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ModelHandler) GetSpatialTree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	tree, err := h.modelSvc.GetSpatialTree(modelID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get spatial tree")
		return
	}
	if tree == nil {
		tree = []*model.SpatialNode{}
	}
	writeJSON(w, http.StatusOK, tree)
}

func (h *ModelHandler) GetElement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	elementID := vars["elementId"]

	element, err := h.modelSvc.GetElement(elementID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get element")
		return
	}
	if element == nil {
		writeError(w, http.StatusNotFound, "Element not found")
		return
	}
	writeJSON(w, http.StatusOK, element)
}

func (h *ModelHandler) GetElementsByType(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]
	category := r.URL.Query().Get("category")

	if category == "" {
		elements, err := h.modelSvc.GetElementsByModel(modelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get elements")
			return
		}
		if elements == nil {
			elements = []*model.Element{}
		}
		writeJSON(w, http.StatusOK, elements)
		return
	}

	elements, err := h.modelSvc.GetElementsByCategory(modelID, category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get elements by category")
		return
	}
	if elements == nil {
		elements = []*model.Element{}
	}
	writeJSON(w, http.StatusOK, elements)
}

func (h *ModelHandler) GetMeshChunk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]
	lodStr := vars["lod"]

	var lod int
	fmt.Sscanf(lodStr, "%d", &lod)

	nodeIDsStr := r.URL.Query().Get("nodes")
	var nodeIDs []string
	if nodeIDsStr != "" {
		nodeIDs = strings.Split(nodeIDsStr, ",")
	}

	chunks, err := h.modelSvc.GetMeshChunks(modelID, lod, nodeIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get mesh chunks")
		return
	}
	if chunks == nil {
		chunks = []*model.MeshChunk{}
	}
	writeJSON(w, http.StatusOK, chunks)
}

func (h *ModelHandler) GetOctree(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modelID := vars["id"]

	writeJSON(w, http.StatusOK, map[string]string{"modelId": modelID, "status": "octree data available via mesh chunks"})
}

func generateUUID() string {
	return uuid.New().String()
}

var _ = os.DevNull
