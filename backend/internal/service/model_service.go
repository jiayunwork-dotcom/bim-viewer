package service

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
		"Penetration/Distance (mm)", "Severity", "Status",
	})

	statusMap := map[model.CollisionStatus]string{
		model.CollisionStatusPending:   "待处理",
		model.CollisionStatusConfirmed: "已确认",
		model.CollisionStatusFalse:     "误报",
		model.CollisionStatusResolved:  "已解决",
	}

	for _, r := range results {
		status := statusMap[r.Status]
		if status == "" {
			status = string(r.Status)
		}
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
			status,
		})
	}

	return nil
}

func (ms *ModelService) GetCollisionStats(taskID string) (*model.CollisionStats, error) {
	return ms.repo.GetCollisionStats(taskID)
}

func (ms *ModelService) GetCollisionStatsByModel(modelID string) (*model.CollisionStats, error) {
	return ms.repo.GetCollisionStatsByModel(modelID)
}

func (ms *ModelService) GetCollisionResultsByModel(modelID string) ([]*model.CollisionResult, error) {
	return ms.repo.GetCollisionResultsByModel(modelID)
}

func (ms *ModelService) GetCollisionHistory(resultID string) ([]*model.CollisionResultHistory, error) {
	return ms.repo.GetCollisionHistory(resultID)
}

func (ms *ModelService) UpdateCollisionResultStatus(resultID string, newStatus model.CollisionStatus, remark string, operator string) error {
	if remark == "" {
		return fmt.Errorf("remark is required")
	}
	validStatuses := map[model.CollisionStatus]bool{
		model.CollisionStatusPending:   true,
		model.CollisionStatusConfirmed: true,
		model.CollisionStatusFalse:     true,
		model.CollisionStatusResolved:  true,
	}
	if !validStatuses[newStatus] {
		return fmt.Errorf("invalid status: %s", newStatus)
	}
	if operator == "" {
		operator = "system"
	}
	return ms.repo.UpdateCollisionResultStatus(resultID, newStatus, remark, operator)
}

func (ms *ModelService) BatchUpdateCollisionStatus(resultIDs []string, newStatus model.CollisionStatus, remark string, operator string) (int, error) {
	if remark == "" {
		return 0, fmt.Errorf("remark is required")
	}
	validStatuses := map[model.CollisionStatus]bool{
		model.CollisionStatusPending:   true,
		model.CollisionStatusConfirmed: true,
		model.CollisionStatusFalse:     true,
		model.CollisionStatusResolved:  true,
	}
	if !validStatuses[newStatus] {
		return 0, fmt.Errorf("invalid status: %s", newStatus)
	}
	if operator == "" {
		operator = "system"
	}
	return ms.repo.BatchUpdateCollisionStatus(resultIDs, newStatus, remark, operator)
}

func (ms *ModelService) GetCollisionTasksByModel(modelID string) ([]*model.CollisionTask, error) {
	rows, err := ms.repo.GetDB().Query(
		`SELECT id, model_id, group_a, group_b, threshold, status, created_at, completed_at 
		 FROM collision_tasks WHERE model_id = $1 ORDER BY created_at DESC`,
		modelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*model.CollisionTask
	for rows.Next() {
		t := &model.CollisionTask{}
		var groupA, groupB string
		var completedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.ModelID, &groupA, &groupB, &t.Threshold, &t.Status, &t.CreatedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(groupA), &t.GroupA)
		json.Unmarshal([]byte(groupB), &t.GroupB)
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
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

func (ms *ModelService) hashElementProperties(e *model.Element) string {
	props := map[string]interface{}{
		"name":         e.Name,
		"type":         e.Type,
		"category":     e.Category,
		"aabbMin":      e.AABBMin,
		"aabbMax":      e.AABBMax,
		"properties":   e.Properties,
		"geometryHash": e.GeometryHash,
	}
	
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		if b, err := json.Marshal(props[k]); err == nil {
			h.Write(b)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (ms *ModelService) CreateVersion(modelID, description string) (*model.ModelVersion, error) {
	elements, err := ms.repo.GetElementsByModel(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get elements: %w", err)
	}

	snapshot := make(map[string]string)
	for _, e := range elements {
		snapshot[e.ID] = ms.hashElementProperties(e)
	}

	versionNumber, err := ms.repo.GetNextVersionNumber(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	versionID := generateUUID()
	version := &model.ModelVersion{
		ID:              versionID,
		ModelID:         modelID,
		VersionNumber:   versionNumber,
		Description:     description,
		ElementSnapshot: snapshot,
	}

	if err := ms.repo.CreateVersion(version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return ms.repo.GetVersion(versionID)
}

func (ms *ModelService) GetVersion(versionID string) (*model.ModelVersion, error) {
	return ms.repo.GetVersion(versionID)
}

func (ms *ModelService) ListVersions(modelID string) ([]*model.ModelVersion, error) {
	return ms.repo.ListVersions(modelID)
}

func (ms *ModelService) DeleteVersion(versionID string) error {
	return ms.repo.DeleteVersion(versionID)
}

func (ms *ModelService) CompareVersions(baseVersionID, compareVersionID string) (*model.VersionDiffResult, error) {
	baseVersion, err := ms.repo.GetVersion(baseVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base version: %w", err)
	}
	if baseVersion == nil {
		return nil, fmt.Errorf("base version not found")
	}

	compareVersion, err := ms.repo.GetVersion(compareVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compare version: %w", err)
	}
	if compareVersion == nil {
		return nil, fmt.Errorf("compare version not found")
	}

	if baseVersion.ModelID != compareVersion.ModelID {
		return nil, fmt.Errorf("versions must belong to the same model")
	}

	diff := &model.VersionDiff{
		Added:     []string{},
		Removed:   []string{},
		Modified:  []string{},
		Unchanged: []string{},
	}

	baseIDs := make(map[string]bool)
	for id := range baseVersion.ElementSnapshot {
		baseIDs[id] = true
	}

	compareIDs := make(map[string]bool)
	for id := range compareVersion.ElementSnapshot {
		compareIDs[id] = true
	}

	for id := range compareVersion.ElementSnapshot {
		if !baseIDs[id] {
			diff.Added = append(diff.Added, id)
		}
	}

	for id := range baseVersion.ElementSnapshot {
		if !compareIDs[id] {
			diff.Removed = append(diff.Removed, id)
		}
	}

	for id := range compareVersion.ElementSnapshot {
		if baseIDs[id] {
			if baseVersion.ElementSnapshot[id] != compareVersion.ElementSnapshot[id] {
				diff.Modified = append(diff.Modified, id)
			} else {
				diff.Unchanged = append(diff.Unchanged, id)
			}
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Modified)
	sort.Strings(diff.Unchanged)

	allElementIDs := make([]string, 0, len(diff.Added)+len(diff.Removed)+len(diff.Modified)+len(diff.Unchanged))
	allElementIDs = append(allElementIDs, diff.Added...)
	allElementIDs = append(allElementIDs, diff.Removed...)
	allElementIDs = append(allElementIDs, diff.Modified...)
	allElementIDs = append(allElementIDs, diff.Unchanged...)

	elements, err := ms.repo.GetElementsByIDs(allElementIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get elements: %w", err)
	}

	elementMap := make(map[string]*model.Element)
	for _, e := range elements {
		elementMap[e.ID] = e
	}

	return &model.VersionDiffResult{
		BaseVersion:     baseVersion,
		CompareVersion:  compareVersion,
		Diff:            diff,
		BaseElements:    elementMap,
		CompareElements: elementMap,
	}, nil
}

func (ms *ModelService) GetVersionElement(versionID, elementID string) (*model.Element, error) {
	version, err := ms.repo.GetVersion(versionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	if version == nil {
		return nil, fmt.Errorf("version not found")
	}

	if _, exists := version.ElementSnapshot[elementID]; !exists {
		return nil, nil
	}

	return ms.repo.GetElement(elementID)
}

func (ms *ModelService) CreateVersionAnnotation(req *model.CreateVersionAnnotationRequest) (*model.VersionAnnotation, error) {
	trimmedContent := strings.TrimSpace(req.Content)
	if trimmedContent == "" {
		return nil, fmt.Errorf("批注内容不能为空")
	}
	if len(trimmedContent) > 500 {
		return nil, fmt.Errorf("批注内容不能超过500字符")
	}

	baseVersion, err := ms.repo.GetVersion(req.BaseVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base version: %w", err)
	}
	if baseVersion == nil {
		return nil, fmt.Errorf("base version not found")
	}

	compareVersion, err := ms.repo.GetVersion(req.CompareVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compare version: %w", err)
	}
	if compareVersion == nil {
		return nil, fmt.Errorf("compare version not found")
	}

	if baseVersion.ModelID != compareVersion.ModelID {
		return nil, fmt.Errorf("versions must belong to the same model")
	}

	author := req.Author
	if author == "" {
		author = "anonymous"
	}

	annotation := &model.VersionAnnotation{
		ID:               generateServiceUUID(),
		BaseVersionID:    req.BaseVersionID,
		CompareVersionID: req.CompareVersionID,
		ElementID:        req.ElementID,
		Content:          trimmedContent,
		Author:           author,
	}

	if err := ms.repo.CreateVersionAnnotation(annotation); err != nil {
		return nil, fmt.Errorf("failed to create version annotation: %w", err)
	}

	return ms.repo.GetVersionAnnotation(annotation.ID)
}

func (ms *ModelService) GetVersionAnnotation(id string) (*model.VersionAnnotation, error) {
	return ms.repo.GetVersionAnnotation(id)
}

func (ms *ModelService) ListVersionAnnotations(baseVersionID, compareVersionID string) ([]*model.VersionAnnotation, error) {
	return ms.repo.ListVersionAnnotations(baseVersionID, compareVersionID)
}

func (ms *ModelService) DeleteVersionAnnotation(id, currentUser string) error {
	author, err := ms.repo.GetVersionAnnotationAuthor(id)
	if err != nil {
		return fmt.Errorf("failed to get annotation author: %w", err)
	}
	if author == "" {
		return fmt.Errorf("annotation not found")
	}

	if currentUser != "" && author != currentUser && author != "anonymous" {
		return fmt.Errorf("permission denied: only author can delete this annotation")
	}

	return ms.repo.DeleteVersionAnnotation(id)
}

func (ms *ModelService) GenerateVersionCompareReport(baseVersionID, compareVersionID string) (*model.VersionCompareReport, error) {
	diffResult, err := ms.CompareVersions(baseVersionID, compareVersionID)
	if err != nil {
		return nil, err
	}

	baseVersion := diffResult.BaseVersion
	compareVersion := diffResult.CompareVersion

	modelInfo, err := ms.repo.GetModel(baseVersion.ModelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info: %w", err)
	}

	annotations, err := ms.repo.ListVersionAnnotations(baseVersionID, compareVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list annotations: %w", err)
	}

	annotationList := []model.VersionAnnotation{}
	for _, a := range annotations {
		annotationList = append(annotationList, *a)
	}

	diff := diffResult.Diff
	changedElements := []model.ReportChangedElement{}

	changeTypeMap := map[string]string{}
	for _, id := range diff.Added {
		changeTypeMap[id] = "added"
	}
	for _, id := range diff.Removed {
		changeTypeMap[id] = "removed"
	}
	for _, id := range diff.Modified {
		changeTypeMap[id] = "modified"
	}

	allChangedIDs := append(append(diff.Added, diff.Removed...), diff.Modified...)
	for _, elementID := range allChangedIDs {
		changeType := changeTypeMap[elementID]
		baseElement := diffResult.BaseElements[elementID]
		compareElement := diffResult.CompareElements[elementID]

		elementName := ""
		elementType := ""
		if compareElement != nil {
			elementName = compareElement.Name
			elementType = compareElement.Type
		} else if baseElement != nil {
			elementName = baseElement.Name
			elementType = baseElement.Type
		}

		propertyDiffs := []model.ReportPropertyDiff{}
		if changeType == "modified" && baseElement != nil && compareElement != nil {
			baseProps := map[string]interface{}{
				"名称":  baseElement.Name,
				"类型":  baseElement.Type,
				"分类":  baseElement.Category,
				"楼层":  baseElement.FloorName,
				"几何哈希": baseElement.GeometryHash,
			}
			compareProps := map[string]interface{}{
				"名称":  compareElement.Name,
				"类型":  compareElement.Type,
				"分类":  compareElement.Category,
				"楼层":  compareElement.FloorName,
				"几何哈希": compareElement.GeometryHash,
			}
			for k, v := range baseElement.Properties {
				baseProps[k] = v
			}
			for k, v := range compareElement.Properties {
				compareProps[k] = v
			}

			allKeys := map[string]bool{}
			for k := range baseProps {
				allKeys[k] = true
			}
			for k := range compareProps {
				allKeys[k] = true
			}

			for key := range allKeys {
				baseVal := baseProps[key]
				compareVal := compareProps[key]
				baseStr := fmt.Sprintf("%v", baseVal)
				compareStr := fmt.Sprintf("%v", compareVal)
				if baseStr != compareStr {
					propertyDiffs = append(propertyDiffs, model.ReportPropertyDiff{
						PropertyName: key,
						BaseValue:    baseVal,
						CompareValue: compareVal,
					})
				}
			}
		}

		changedElements = append(changedElements, model.ReportChangedElement{
			ElementID:     elementID,
			ElementName:   elementName,
			ElementType:   elementType,
			ChangeType:    changeType,
			PropertyDiffs: propertyDiffs,
		})
	}

	report := &model.VersionCompareReport{
		MetaInfo: model.ReportMetaInfo{
			ModelName:         modelInfo.Name,
			BaseVersion:       baseVersion.VersionNumber,
			CompareVersion:    compareVersion.VersionNumber,
			BaseVersionDesc:   baseVersion.Description,
			CompareVersionDesc: compareVersion.Description,
			CompareTime:       time.Now(),
		},
		DiffStats: model.ReportDiffStats{
			Added:     len(diff.Added),
			Removed:   len(diff.Removed),
			Modified:  len(diff.Modified),
			Unchanged: len(diff.Unchanged),
			Total:     len(diff.Added) + len(diff.Removed) + len(diff.Modified) + len(diff.Unchanged),
		},
		ChangedElements: changedElements,
		Annotations:     annotationList,
	}

	return report, nil
}

func generateServiceUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
