package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"bim-viewer/internal/model"
	"bim-viewer/internal/repository"
)

type ModelService struct {
	repo      *repository.PostgresRepo
	ifcParser *IFCParserService
}

func NewModelService(repo *repository.PostgresRepo, ifcParser *IFCParserService) *ModelService {
	return &ModelService{
		repo:      repo,
		ifcParser: ifcParser,
	}
}

func (ms *ModelService) UploadModel(fileName string, fileSize int64, reader io.Reader) (*model.Model, error) {
	modelID := generateUUID()

	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, fmt.Sprintf("ifc_%s%s", modelID, filepath.Ext(fileName)))

	f, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	m := &model.Model{
		ID:        modelID,
		Name:      strings.TrimSuffix(fileName, filepath.Ext(fileName)),
		FileName:  fileName,
		FileSize:  fileSize,
		Status:    "uploading",
	}

	if err := ms.repo.CreateModel(m); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to create model record: %w", err)
	}

	ms.repo.UpdateModelStatus(modelID, "parsing")

	parsed, err := ms.ifcParser.ParseIFC(tmpPath, modelID, fileName)
	if err != nil {
		ms.repo.UpdateModelStatus(modelID, "error")
		os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to parse IFC: %w", err)
	}

	for _, node := range flattenSpatialTree(parsed.SpatialTree) {
		if err := ms.repo.CreateSpatialNode(node); err != nil {
			fmt.Printf("Warning: failed to create spatial node: %v\n", err)
		}
	}

	for _, element := range parsed.Elements {
		if err := ms.repo.CreateElement(element); err != nil {
			fmt.Printf("Warning: failed to create element: %v\n", err)
		}
	}

	for _, chunk := range parsed.MeshChunks {
		if err := ms.repo.CreateMeshChunk(chunk); err != nil {
			fmt.Printf("Warning: failed to create mesh chunk: %v\n", err)
		}
	}

	if err := ms.repo.UpdateModelStats(modelID, parsed.Model.TriangleCount, parsed.Model.ElementCount); err != nil {
		fmt.Printf("Warning: failed to update model stats: %v\n", err)
	}

	ms.repo.UpdateModelStatus(modelID, "ready")

	os.Remove(tmpPath)

	result, _ := ms.repo.GetModel(modelID)
	return result, nil
}

func (ms *ModelService) GetModel(id string) (*model.Model, error) {
	return ms.repo.GetModel(id)
}

func (ms *ModelService) ListModels() ([]*model.Model, error) {
	return ms.repo.ListModels()
}

func (ms *ModelService) DeleteModel(id string) error {
	return ms.repo.DeleteModel(id)
}

func (ms *ModelService) GetSpatialTree(modelID string) ([]*model.SpatialNode, error) {
	return ms.repo.GetSpatialTree(modelID)
}

func (ms *ModelService) GetElement(id string) (*model.Element, error) {
	return ms.repo.GetElement(id)
}

func (ms *ModelService) GetElementsByCategory(modelID, category string) ([]*model.Element, error) {
	return ms.repo.GetElementsByCategory(modelID, category)
}

func (ms *ModelService) GetMeshChunks(modelID string, lod int, nodeIDs []string) ([]*model.MeshChunk, error) {
	return ms.repo.GetMeshChunks(modelID, lod, nodeIDs)
}

func (ms *ModelService) GetElementsByModel(modelID string) ([]*model.Element, error) {
	return ms.repo.GetElementsByModel(modelID)
}

func (ms *ModelService) RunCollisionDetection(req CollisionRequest) (string, []*model.CollisionResult, error) {
	taskID := generateUUID()

	task := &model.CollisionTask{
		ID:        taskID,
		ModelID:   req.ModelID,
		GroupA:    req.GroupA,
		GroupB:    req.GroupB,
		Threshold: req.Threshold,
		Status:    "running",
	}

	if err := ms.repo.CreateCollisionTask(task); err != nil {
		return "", nil, fmt.Errorf("failed to create collision task: %w", err)
	}

	results, err := NewCollisionService().DetectCollisions(ms.repo, req)
	if err != nil {
		ms.repo.UpdateCollisionTaskStatus(taskID, "error")
		return taskID, nil, err
	}

	for _, r := range results {
		r.TaskID = taskID
		if err := ms.repo.CreateCollisionResult(r); err != nil {
			fmt.Printf("Warning: failed to save collision result: %v\n", err)
		}
	}

	ms.repo.UpdateCollisionTaskStatus(taskID, "completed")

	return taskID, results, nil
}

func (ms *ModelService) GetCollisionResults(taskID string) ([]*model.CollisionResult, error) {
	return ms.repo.GetCollisionResults(taskID)
}

func (ms *ModelService) ExportCollisionCSV(taskID string, writer io.Writer) error {
	results, err := ms.repo.GetCollisionResults(taskID)
	if err != nil {
		return err
	}

	w := csv.NewWriter(writer)
	defer w.Flush()

	w.Write([]string{
		"ID", "Element A Name", "Element A Type", "Element A Floor",
		"Element B Name", "Element B Type", "Element B Floor",
		"Collision Type", "Collision Point X", "Collision Point Y", "Collision Point Z",
		"Penetration/Distance (mm)", "Severity",
	})

	for _, r := range results {
		w.Write([]string{
			r.ID,
			r.ElementAName, r.ElementAType, r.ElementAFloor,
			r.ElementBName, r.ElementBType, r.ElementBFloor,
			r.CollisionType,
			fmt.Sprintf("%.2f", r.CollisionPoint[0]),
			fmt.Sprintf("%.2f", r.CollisionPoint[1]),
			fmt.Sprintf("%.2f", r.CollisionPoint[2]),
			fmt.Sprintf("%.2f", r.Penetration),
			r.Severity,
		})
	}

	return nil
}

func flattenSpatialTree(nodes []*model.SpatialNode) []*model.SpatialNode {
	var result []*model.SpatialNode
	var traverse func(n *model.SpatialNode)
	traverse = func(n *model.SpatialNode) {
		result = append(result, n)
		for _, c := range n.Children {
			traverse(c)
		}
	}
	for _, n := range nodes {
		traverse(n)
	}
	return result
}
