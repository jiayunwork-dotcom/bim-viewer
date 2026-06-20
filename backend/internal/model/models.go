package model

import "time"

type Model struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	FileName    string    `json:"fileName"`
	IFCVersion  string    `json:"ifcVersion"`
	FileSize    int64     `json:"fileSize"`
	Status      string    `json:"status"`
	TriangleCount int     `json:"triangleCount"`
	ElementCount  int     `json:"elementCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SpatialNode struct {
	ID         string         `json:"id"`
	ModelID    string         `json:"modelId"`
	ParentID   *string        `json:"parentId,omitempty"`
	IFCGUID    string         `json:"ifcGuid"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Level      int            `json:"level"`
	Children   []*SpatialNode `json:"children,omitempty"`
}

type Element struct {
	ID           string                 `json:"id"`
	ModelID      string                 `json:"modelId"`
	IFCGUID      string                 `json:"ifcGuid"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Category     string                 `json:"category"`
	ParentID     *string                `json:"parentId,omitempty"`
	FloorName    string                 `json:"floorName,omitempty"`
	AABBMin      [3]float64            `json:"aabbMin"`
	AABBMax      [3]float64            `json:"aabbMax"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
	MeshLODs     map[string]string      `json:"meshLods,omitempty"`
	GeometryHash string                 `json:"geometryHash,omitempty"`
}

type MeshChunk struct {
	ID           string  `json:"id"`
	ModelID      string  `json:"modelId"`
	LOD          int     `json:"lod"`
	OctreeNodeID string  `json:"octreeNodeId"`
	Data         []byte  `json:"data"`
	VertexCount  int     `json:"vertexCount"`
	IndexCount   int     `json:"indexCount"`
}

type OctreeNode struct {
	ID       string         `json:"id"`
	ModelID  string         `json:"modelId"`
	Center   [3]float64    `json:"center"`
	HalfSize float64        `json:"halfSize"`
	Depth    int            `json:"depth"`
	Children []*OctreeNode  `json:"children,omitempty"`
	Elements []string       `json:"elements,omitempty"`
}

type CollisionTask struct {
	ID          string    `json:"id"`
	ModelID     string    `json:"modelId"`
	GroupA      []string  `json:"groupA"`
	GroupB      []string  `json:"groupB"`
	Threshold   float64   `json:"threshold"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type CollisionStatus string

const (
	CollisionStatusPending   CollisionStatus = "pending"
	CollisionStatusConfirmed CollisionStatus = "confirmed"
	CollisionStatusFalse     CollisionStatus = "false_positive"
	CollisionStatusResolved  CollisionStatus = "resolved"
)

type CollisionResult struct {
	ID              string          `json:"id"`
	TaskID          string          `json:"taskId"`
	ElementAID      string          `json:"elementAId"`
	ElementAName    string          `json:"elementAName"`
	ElementAType    string          `json:"elementAType"`
	ElementAFloor   string          `json:"elementAFloor"`
	ElementBID      string          `json:"elementBId"`
	ElementBName    string          `json:"elementBName"`
	ElementBType    string          `json:"elementBType"`
	ElementBFloor   string          `json:"elementBFloor"`
	CollisionType   string          `json:"collisionType"`
	CollisionPoint  [3]float64      `json:"collisionPoint"`
	Penetration     float64         `json:"penetration"`
	Severity        string          `json:"severity"`
	Status          CollisionStatus `json:"status"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type CollisionResultHistory struct {
	ID            string          `json:"id"`
	ResultID      string          `json:"resultId"`
	TaskID        string          `json:"taskId"`
	OldStatus     CollisionStatus `json:"oldStatus"`
	NewStatus     CollisionStatus `json:"newStatus"`
	Remark        string          `json:"remark"`
	Operator      string          `json:"operator"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type CollisionStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Confirmed int `json:"confirmed"`
	False     int `json:"false_positive"`
	Resolved  int `json:"resolved"`
}

type UpdateStatusRequest struct {
	ResultIDs []string        `json:"resultIds"`
	NewStatus CollisionStatus `json:"newStatus"`
	Remark    string          `json:"remark"`
	Operator  string          `json:"operator"`
}

type ModelVersion struct {
	ID              string                 `json:"id"`
	ModelID         string                 `json:"modelId"`
	VersionNumber   string                 `json:"versionNumber"`
	Description     string                 `json:"description"`
	ElementSnapshot map[string]string      `json:"elementSnapshot"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type VersionDiff struct {
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Modified []string `json:"modified"`
	Unchanged []string `json:"unchanged"`
}

type VersionDiffResult struct {
	BaseVersion     *ModelVersion `json:"baseVersion"`
	CompareVersion  *ModelVersion `json:"compareVersion"`
	Diff            *VersionDiff  `json:"diff"`
	BaseElements    map[string]*Element `json:"baseElements,omitempty"`
	CompareElements map[string]*Element `json:"compareElements,omitempty"`
}

type CreateVersionRequest struct {
	Description string `json:"description"`
}

type CompareVersionsRequest struct {
	BaseVersionID    string `json:"baseVersionId"`
	CompareVersionID string `json:"compareVersionId"`
}
