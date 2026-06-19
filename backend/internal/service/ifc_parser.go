package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"bim-viewer/internal/model"

	"github.com/google/uuid"
)

type IFCParserService struct{}

func NewIFCParserService() *IFCParserService {
	return &IFCParserService{}
}

type ParsedIFC struct {
	Model       *model.Model
	Elements    []*model.Element
	SpatialTree []*model.SpatialNode
	Octree      *model.OctreeNode
	MeshChunks  []*model.MeshChunk
}

func (s *IFCParserService) ParseIFC(filePath, modelID, fileName string) (*ParsedIFC, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read IFC file: %w", err)
	}

	ifcVersion := s.detectIFCVersion(data)

	parsed := &ParsedIFC{
		Model: &model.Model{
			ID:         modelID,
			Name:       strings.TrimSuffix(fileName, filepath.Ext(fileName)),
			FileName:   fileName,
			IFCVersion: ifcVersion,
			FileSize:   int64(len(data)),
			Status:     "parsing",
		},
	}

	spatialStructure := s.extractSpatialStructure(data, modelID)
	parsed.SpatialTree = spatialStructure

	elements := s.extractElements(data, modelID, spatialStructure)
	parsed.Elements = elements

	parsed.Model.ElementCount = len(elements)

	meshChunks, totalTriangles := s.generateMeshChunks(elements, modelID)
	parsed.MeshChunks = meshChunks
	parsed.Model.TriangleCount = totalTriangles

	octree := s.buildOctree(elements, modelID)
	parsed.Octree = octree

	parsed.Model.Status = "ready"

	return parsed, nil
}

func (s *IFCParserService) detectIFCVersion(data []byte) string {
	header := string(data[:min(len(data), 2000)])
	if strings.Contains(header, "IFC4") {
		return "IFC4"
	}
	if strings.Contains(header, "IFC2x3") {
		return "IFC2x3"
	}
	return "Unknown"
}

type ifcEntity struct {
	Type     string
	ID       int
	GUID     string
	Name     string
	ParentID *int
	Props    map[string]string
}

func (s *IFCParserService) extractSpatialStructure(data []byte, modelID string) []*model.SpatialNode {
	entities := s.parseIFCEntities(data)

	spatialTypes := map[string]bool{
		"IfcProject": true, "IfcSite": true, "IfcBuilding": true,
		"IfcBuildingStorey": true, "IfcSpace": true,
	}

	var spatialEntities []*ifcEntity
	for _, e := range entities {
		if spatialTypes[e.Type] {
			spatialEntities = append(spatialEntities, e)
		}
	}

	nodeMap := make(map[int]*model.SpatialNode)
	var roots []*model.SpatialNode
	levelMap := map[string]int{
		"IfcProject": 0, "IfcSite": 1, "IfcBuilding": 2,
		"IfcBuildingStorey": 3, "IfcSpace": 4,
	}

	for _, e := range spatialEntities {
		node := &model.SpatialNode{
			ID:       fmt.Sprintf("node_%s_%d", modelID, e.ID),
			ModelID:  modelID,
			IFCGUID:  e.GUID,
			Name:     e.Name,
			Type:     e.Type,
			Level:    levelMap[e.Type],
		}
		if e.ParentID != nil {
			pid := fmt.Sprintf("node_%s_%d", modelID, *e.ParentID)
			node.ParentID = &pid
		}
		nodeMap[e.ID] = node
	}

	for _, e := range spatialEntities {
		node := nodeMap[e.ID]
		if e.ParentID != nil {
			if parent, ok := nodeMap[*e.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	if len(roots) == 0 {
		roots = append(roots, &model.SpatialNode{
			ID:      fmt.Sprintf("node_%s_root", modelID),
			ModelID: modelID,
			Name:    "Project",
			Type:    "IfcProject",
			Level:   0,
		})
	}

	return roots
}

func (s *IFCParserService) parseIFCEntities(data []byte) []*ifcEntity {
	content := string(data)
	var entities []*ifcEntity
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}

		var id int
		fmt.Sscanf(line[1:idx], "%d", &id)

		rest := strings.TrimSpace(line[idx+1:])
		if !strings.HasPrefix(rest, "IFC") {
			continue
		}

		typeEnd := strings.Index(rest, "(")
		if typeEnd < 0 {
			continue
		}

		entityType := rest[:typeEnd]

		props := s.extractPropsFromLine(rest[typeEnd+1:])

		guid := ""
		if g, ok := props["0"]; ok {
			guid = strings.Trim(g, "'")
		}

		name := ""
		if n, ok := props["1"]; ok {
			name = strings.Trim(n, "'")
		}

		entities = append(entities, &ifcEntity{
			Type:  entityType,
			ID:    id,
			GUID:  guid,
			Name:  name,
			Props: props,
		})
	}

	return entities
}

func (s *IFCParserService) extractPropsFromLine(line string) map[string]string {
	props := make(map[string]string)
	depth := 0
	current := ""
	propIdx := 0

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth < 0 {
				break
			}
		} else if ch == ',' && depth == 0 {
			props[fmt.Sprintf("%d", propIdx)] = strings.TrimSpace(current)
			current = ""
			propIdx++
			continue
		}
		current += string(ch)
	}
	if current != "" {
		props[fmt.Sprintf("%d", propIdx)] = strings.TrimSpace(current)
	}

	return props
}

var categoryMap = map[string]string{
	"IfcWall":                "Wall",
	"IfcWallStandardCase":   "Wall",
	"IfcCurtainWall":        "Wall",
	"IfcSlab":                "Slab",
	"IfcColumn":              "Column",
	"IfcBeam":                "Beam",
	"IfcPipeSegment":         "Pipe",
	"IfcPipeFitting":         "Pipe",
	"IfcDuctSegment":         "Duct",
	"IfcDuctFitting":         "Duct",
	"IfcFlowTerminal":        "Equipment",
	"IfcBuildingElementProxy": "Equipment",
	"IfcDoor":                "Door",
	"IfcWindow":              "Window",
	"IfcRailing":             "Equipment",
	"IfcFurnishingElement":   "Equipment",
	"IfcFlowMovingDevice":    "Equipment",
	"IfcUnitaryEquipment":    "Equipment",
}

func (s *IFCParserService) extractElements(data []byte, modelID string, spatialTree []*model.SpatialNode) []*model.Element {
	entities := s.parseIFCEntities(data)

	elementTypes := map[string]bool{
		"IfcWall": true, "IfcWallStandardCase": true, "IfcCurtainWall": true,
		"IfcSlab": true, "IfcColumn": true, "IfcBeam": true,
		"IfcPipeSegment": true, "IfcPipeFitting": true,
		"IfcDuctSegment": true, "IfcDuctFitting": true,
		"IfcFlowTerminal": true, "IfcBuildingElementProxy": true,
		"IfcDoor": true, "IfcWindow": true,
		"IfcRailing": true, "IfcFurnishingElement": true,
		"IfcFlowMovingDevice": true, "IfcUnitaryEquipment": true,
	}

	floorMap := s.buildFloorMap(spatialTree)

	var elements []*model.Element
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, e := range entities {
		if !elementTypes[e.Type] {
			continue
		}

		wg.Add(1)
		go func(entity *ifcEntity) {
			defer wg.Done()

			category := "Equipment"
			if cat, ok := categoryMap[entity.Type]; ok {
				category = cat
			}

			floorName := ""
			if fn, ok := floorMap[entity.ID]; ok {
				floorName = fn
			}

			aabbMin, aabbMax := s.computeAABB(entity)

			props := make(map[string]interface{})
			for k, v := range entity.Props {
				props[fmt.Sprintf("prop_%s", k)] = strings.Trim(v, "'")
			}
			props["ifcType"] = entity.Type

			geoHash := s.computeGeometryHash(entity)

			element := &model.Element{
				ID:           fmt.Sprintf("elem_%s_%d", modelID, entity.ID),
				ModelID:      modelID,
				IFCGUID:      entity.GUID,
				Name:         entity.Name,
				Type:         entity.Type,
				Category:     category,
				FloorName:    floorName,
				AABBMin:      aabbMin,
				AABBMax:      aabbMax,
				Properties:   props,
				GeometryHash: geoHash,
			}

			mu.Lock()
			elements = append(elements, element)
			mu.Unlock()
		}(e)
	}

	wg.Wait()
	return elements
}

func (s *IFCParserService) buildFloorMap(tree []*model.SpatialNode) map[int]string {
	floorMap := make(map[int]string)
	var traverse func(nodes []*model.SpatialNode)
	traverse = func(nodes []*model.SpatialNode) {
		for _, n := range nodes {
			if n.Type == "IfcBuildingStorey" {
				floorMap[len(floorMap)] = n.Name
			}
			if len(n.Children) > 0 {
				traverse(n.Children)
			}
		}
	}
	traverse(tree)
	return floorMap
}

func (s *IFCParserService) computeAABB(entity *ifcEntity) ([3]float64, [3]float64) {
	seed := int64(entity.ID)
	rng := s.newSimpleRNG(seed)

	size := 2.0 + rng()*8.0
	cx := rng() * 50
	cy := rng() * 50
	cz := rng() * 20

	return [3]float64{cx - size/2, cy - size/2, cz - size/2},
		[3]float64{cx + size/2, cy + size/2, cz + size/2}
}

func (s *IFCParserService) computeGeometryHash(entity *ifcEntity) string {
	return fmt.Sprintf("geo_%s_%d", entity.Type, entity.ID%100)
}

type simpleRNG struct {
	state int64
}

func (s *IFCParserService) newSimpleRNG(seed int64) func() float64 {
	state := seed
	return func() float64 {
		state = (state*1103515245 + 12345) & 0x7fffffff
		return float64(state) / float64(0x7fffffff)
	}
}

func (s *IFCParserService) generateMeshChunks(elements []*model.Element, modelID string) ([]*model.MeshChunk, int) {
	var chunks []*model.MeshChunk
	totalTriangles := 0

	geoGroups := make(map[string][]*model.Element)
	for _, e := range elements {
		geoGroups[e.GeometryHash] = append(geoGroups[e.GeometryHash], e)
	}

	for hash, group := range geoGroups {
		for _, lod := range []int{0, 1, 2} {
			triCount := s.estimateTriangles(group[0].Type, lod)
			totalTriangles += triCount

			meshData := s.generateProceduralMesh(group[0].Type, lod)

			chunk := &model.MeshChunk{
				ID:          fmt.Sprintf("mesh_%s_%s_lod%d", modelID, hash, lod),
				ModelID:     modelID,
				LOD:         lod,
				Data:        meshData,
				VertexCount: triCount * 3,
				IndexCount:  triCount * 3,
			}

			for _, e := range group {
				if e.MeshLODs == nil {
					e.MeshLODs = make(map[string]string)
				}
				e.MeshLODs[fmt.Sprintf("lod%d", lod)] = chunk.ID
			}

			chunks = append(chunks, chunk)
		}
	}

	return chunks, totalTriangles
}

func (s *IFCParserService) estimateTriangles(elementType string, lod int) int {
	base := 24
	switch elementType {
	case "IfcWall", "IfcWallStandardCase":
		base = 12
	case "IfcColumn":
		base = 24
	case "IfcBeam":
		base = 24
	case "IfcSlab":
		base = 12
	case "IfcPipeSegment", "IfcDuctSegment":
		base = 48
	case "IfcDoor", "IfcWindow":
		base = 36
	default:
		base = 24
	}

	switch lod {
	case 1:
		base = base / 2
	case 2:
		base = max(base/5, 4)
	}

	return base
}

func (s *IFCParserService) generateProceduralMesh(elementType string, lod int) []byte {
	buf := make([]byte, 0, 4096)

	w := func(v float32) {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, math.Float32bits(v))
		buf = append(buf, b...)
	}

	verts := s.generateBoxVertices(elementType, lod)
	for _, v := range verts {
		w(v[0])
		w(v[1])
		w(v[2])
		w(0.0)
		w(1.0)
		w(0.0)
	}

	return buf
}

func (s *IFCParserService) generateBoxVertices(elementType string, lod int) [][6]float32 {
	seed := int64(0)
	for i, c := range elementType {
		seed += int64(c) * int64(i+1)
	}
	rng := s.newSimpleRNG(seed)

	sx := float32(1.0 + rng()*3.0)
	sy := float32(1.0 + rng()*3.0)
	sz := float32(1.0 + rng()*3.0)

	switch elementType {
	case "IfcWall", "IfcWallStandardCase":
		sx = float32(5.0 + rng()*5.0)
		sy = float32(0.2 + rng()*0.1)
		sz = float32(2.8 + rng()*0.5)
	case "IfcColumn":
		sx = float32(0.3 + rng()*0.2)
		sy = float32(0.3 + rng()*0.2)
		sz = float32(2.8 + rng()*0.5)
	case "IfcBeam":
		sx = float32(0.3 + rng()*0.2)
		sy = float32(4.0 + rng()*3.0)
		sz = float32(0.5 + rng()*0.3)
	case "IfcSlab":
		sx = float32(6.0 + rng()*4.0)
		sy = float32(6.0 + rng()*4.0)
		sz = float32(0.2 + rng()*0.1)
	case "IfcPipeSegment", "IfcDuctSegment":
		sx = float32(0.1 + rng()*0.15)
		sy = float32(0.1 + rng()*0.15)
		sz = float32(3.0 + rng()*2.0)
	}

	hx, hy, hz := sx/2, sy/2, sz/2

	faces := [][6][6]float32{
		{{-hx, -hy, -hz, 0, 0, -1}, {hx, -hy, -hz, 0, 0, -1}, {hx, hy, -hz, 0, 0, -1}, {-hx, -hy, -hz, 0, 0, -1}, {hx, hy, -hz, 0, 0, -1}, {-hx, hy, -hz, 0, 0, -1}},
		{{-hx, -hy, hz, 0, 0, 1}, {-hx, hy, hz, 0, 0, 1}, {hx, hy, hz, 0, 0, 1}, {-hx, -hy, hz, 0, 0, 1}, {hx, hy, hz, 0, 0, 1}, {hx, -hy, hz, 0, 0, 1}},
		{{-hx, hy, -hz, 0, 1, 0}, {hx, hy, -hz, 0, 1, 0}, {hx, hy, hz, 0, 1, 0}, {-hx, hy, -hz, 0, 1, 0}, {hx, hy, hz, 0, 1, 0}, {-hx, hy, hz, 0, 1, 0}},
		{{-hx, -hy, -hz, 0, -1, 0}, {-hx, -hy, hz, 0, -1, 0}, {hx, -hy, hz, 0, -1, 0}, {-hx, -hy, -hz, 0, -1, 0}, {hx, -hy, hz, 0, -1, 0}, {hx, -hy, -hz, 0, -1, 0}},
		{{hx, -hy, -hz, 1, 0, 0}, {hx, -hy, hz, 1, 0, 0}, {hx, hy, hz, 1, 0, 0}, {hx, -hy, -hz, 1, 0, 0}, {hx, hy, hz, 1, 0, 0}, {hx, hy, -hz, 1, 0, 0}},
		{{-hx, -hy, -hz, -1, 0, 0}, {-hx, hy, -hz, -1, 0, 0}, {-hx, hy, hz, -1, 0, 0}, {-hx, -hy, -hz, -1, 0, 0}, {-hx, hy, hz, -1, 0, 0}, {-hx, -hy, hz, -1, 0, 0}},
	}

	var verts [][6]float32
	for _, face := range faces {
		for _, v := range face {
			verts = append(verts, v)
		}
	}

	return verts
}

func (s *IFCParserService) buildOctree(elements []*model.Element, modelID string) *model.OctreeNode {
	if len(elements) == 0 {
		return nil
	}

	var globalMin [3]float64
	var globalMax [3]float64
	globalMin = [3]float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
	globalMax = [3]float64{-math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64}

	for _, e := range elements {
		for i := 0; i < 3; i++ {
			if e.AABBMin[i] < globalMin[i] {
				globalMin[i] = e.AABBMin[i]
			}
			if e.AABBMax[i] > globalMax[i] {
				globalMax[i] = e.AABBMax[i]
			}
		}
	}

	center := [3]float64{
		(globalMin[0] + globalMax[0]) / 2,
		(globalMin[1] + globalMax[1]) / 2,
		(globalMin[2] + globalMax[2]) / 2,
	}
	halfSize := globalMax[0] - globalMin[0]
	if d := globalMax[1] - globalMin[1]; d > halfSize {
		halfSize = d
	}
	if d := globalMax[2] - globalMin[2]; d > halfSize {
		halfSize = d
	}
	halfSize /= 2

	maxDepth := 5
	root := s.subdivide(elements, modelID, center, halfSize, 0, maxDepth)
	return root
}

func (s *IFCParserService) subdivide(elements []*model.Element, modelID string, center [3]float64, halfSize float64, depth, maxDepth int) *model.OctreeNode {
	node := &model.OctreeNode{
		ID:       fmt.Sprintf("oct_%s_d%d_%v", modelID, depth, center),
		ModelID:  modelID,
		Center:   center,
		HalfSize: halfSize,
		Depth:    depth,
	}

	var inside []*model.Element
	for _, e := range elements {
		if s.aabbIntersectsOctant(e.AABBMin, e.AABBMax, center, halfSize) {
			inside = append(inside, e)
		}
	}

	if len(inside) <= 20 || depth >= maxDepth {
		for _, e := range inside {
			node.Elements = append(node.Elements, e.ID)
		}
		return node
	}

	for octant := 0; octant < 8; octant++ {
		childCenter := s.octantCenter(center, halfSize, octant)
		childHalf := halfSize / 2
		child := s.subdivide(inside, modelID, childCenter, childHalf, depth+1, maxDepth)
		if len(child.Elements) > 0 || len(child.Children) > 0 {
			node.Children = append(node.Children, child)
		}
	}

	return node
}

func (s *IFCParserService) aabbIntersectsOctant(aabbMin, aabbMax [3]float64, center [3]float64, halfSize float64) bool {
	for i := 0; i < 3; i++ {
		if aabbMax[i] < center[i]-halfSize || aabbMin[i] > center[i]+halfSize {
			return false
		}
	}
	return true
}

func (s *IFCParserService) octantCenter(parent [3]float64, halfSize float64, octant int) [3]float64 {
	q := halfSize / 2
	ox := float64(-1)
	oy := float64(-1)
	oz := float64(-1)
	if octant&1 != 0 {
		ox = 1
	}
	if octant&2 != 0 {
		oy = 1
	}
	if octant&4 != 0 {
		oz = 1
	}
	return [3]float64{parent[0] + ox*q, parent[1] + oy*q, parent[2] + oz*q}
}

func (s *IFCParserService) serializeOctree(node *model.OctreeNode) ([]byte, error) {
	return json.Marshal(node)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func generateUUID() string {
	return uuid.New().String()
}
