package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"bim-viewer/internal/model"

	_ "github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(dbURL string) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) Close() {
	r.db.Close()
}

func (r *PostgresRepo) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS models (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			file_name VARCHAR(255) NOT NULL,
			ifc_version VARCHAR(32),
			file_size BIGINT DEFAULT 0,
			status VARCHAR(32) DEFAULT 'uploading',
			triangle_count INTEGER DEFAULT 0,
			element_count INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS spatial_nodes (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			parent_id VARCHAR(64),
			ifc_guid VARCHAR(128),
			name VARCHAR(255),
			type VARCHAR(64),
			level INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS elements (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			ifc_guid VARCHAR(128),
			name VARCHAR(255),
			type VARCHAR(64),
			category VARCHAR(64),
			parent_id VARCHAR(64),
			floor_name VARCHAR(128),
			aabb_min DOUBLE PRECISION[3],
			aabb_max DOUBLE PRECISION[3],
			properties JSONB,
			mesh_lods JSONB,
			geometry_hash VARCHAR(128)
		)`,
		`CREATE TABLE IF NOT EXISTS mesh_chunks (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			lod INTEGER,
			octree_node_id VARCHAR(64),
			data BYTEA,
			vertex_count INTEGER DEFAULT 0,
			index_count INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS octree_nodes (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			center DOUBLE PRECISION[3],
			half_size DOUBLE PRECISION,
			depth INTEGER DEFAULT 0,
			parent_id VARCHAR(64),
			elements TEXT[]
		)`,
		`CREATE TABLE IF NOT EXISTS collision_tasks (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			group_a TEXT[],
			group_b TEXT[],
			threshold DOUBLE PRECISION DEFAULT 50.0,
			status VARCHAR(32) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS collision_results (
			id VARCHAR(64) PRIMARY KEY,
			task_id VARCHAR(64) REFERENCES collision_tasks(id) ON DELETE CASCADE,
			element_a_id VARCHAR(64),
			element_a_name VARCHAR(255),
			element_a_type VARCHAR(64),
			element_a_floor VARCHAR(128),
			element_b_id VARCHAR(64),
			element_b_name VARCHAR(255),
			element_b_type VARCHAR(64),
			element_b_floor VARCHAR(128),
			collision_type VARCHAR(32),
			collision_point DOUBLE PRECISION[3],
			penetration DOUBLE PRECISION DEFAULT 0,
			severity VARCHAR(16)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_elements_model_id ON elements(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_elements_category ON elements(category)`,
		`CREATE INDEX IF NOT EXISTS idx_elements_geometry_hash ON elements(geometry_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_mesh_chunks_model_lod ON mesh_chunks(model_id, lod)`,
		`CREATE INDEX IF NOT EXISTS idx_collision_results_task ON collision_results(task_id)`,
	}
	for _, m := range migrations {
		if _, err := r.db.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

func (r *PostgresRepo) CreateModel(m *model.Model) error {
	_, err := r.db.Exec(
		`INSERT INTO models (id, name, file_name, ifc_version, file_size, status, triangle_count, element_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		m.ID, m.Name, m.FileName, m.IFCVersion, m.FileSize, m.Status, m.TriangleCount, m.ElementCount,
	)
	return err
}

func (r *PostgresRepo) UpdateModelStatus(id, status string) error {
	_, err := r.db.Exec(`UPDATE models SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (r *PostgresRepo) UpdateModelStats(id string, triangles, elements int) error {
	_, err := r.db.Exec(`UPDATE models SET triangle_count = $2, element_count = $3, updated_at = NOW() WHERE id = $1`, id, triangles, elements)
	return err
}

func (r *PostgresRepo) GetModel(id string) (*model.Model, error) {
	row := r.db.QueryRow(`SELECT id, name, file_name, ifc_version, file_size, status, triangle_count, element_count, created_at, updated_at FROM models WHERE id = $1`, id)
	m := &model.Model{}
	err := row.Scan(&m.ID, &m.Name, &m.FileName, &m.IFCVersion, &m.FileSize, &m.Status, &m.TriangleCount, &m.ElementCount, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

func (r *PostgresRepo) ListModels() ([]*model.Model, error) {
	rows, err := r.db.Query(`SELECT id, name, file_name, ifc_version, file_size, status, triangle_count, element_count, created_at, updated_at FROM models ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []*model.Model
	for rows.Next() {
		m := &model.Model{}
		if err := rows.Scan(&m.ID, &m.Name, &m.FileName, &m.IFCVersion, &m.FileSize, &m.Status, &m.TriangleCount, &m.ElementCount, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *PostgresRepo) DeleteModel(id string) error {
	_, err := r.db.Exec(`DELETE FROM models WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) CreateElement(e *model.Element) error {
	propsJSON, _ := json.Marshal(e.Properties)
	lodsJSON, _ := json.Marshal(e.MeshLODs)
	_, err := r.db.Exec(
		`INSERT INTO elements (id, model_id, ifc_guid, name, type, category, parent_id, floor_name, aabb_min, aabb_max, properties, mesh_lods, geometry_hash)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		e.ID, e.ModelID, e.IFCGUID, e.Name, e.Type, e.Category, e.ParentID, e.FloorName,
		fmt.Sprintf(`[%f,%f,%f]`, e.AABBMin[0], e.AABBMin[1], e.AABBMin[2]),
		fmt.Sprintf(`[%f,%f,%f]`, e.AABBMax[0], e.AABBMax[1], e.AABBMax[2]),
		propsJSON, lodsJSON, e.GeometryHash,
	)
	return err
}

func (r *PostgresRepo) GetElement(id string) (*model.Element, error) {
	row := r.db.QueryRow(`SELECT id, model_id, ifc_guid, name, type, category, parent_id, floor_name, aabb_min, aabb_max, properties, mesh_lods, geometry_hash FROM elements WHERE id = $1`, id)
	return r.scanElement(row)
}

func (r *PostgresRepo) GetElementsByModel(modelID string) ([]*model.Element, error) {
	rows, err := r.db.Query(`SELECT id, model_id, ifc_guid, name, type, category, parent_id, floor_name, aabb_min, aabb_max, properties, mesh_lods, geometry_hash FROM elements WHERE model_id = $1`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var elements []*model.Element
	for rows.Next() {
		e := &model.Element{}
		var aabbMin, aabbMax, props, lods string
		if err := rows.Scan(&e.ID, &e.ModelID, &e.IFCGUID, &e.Name, &e.Type, &e.Category, &e.ParentID, &e.FloorName, &aabbMin, &aabbMax, &props, &lods, &e.GeometryHash); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(props), &e.Properties)
		json.Unmarshal([]byte(lods), &e.MeshLODs)
		elements = append(elements, e)
	}
	return elements, nil
}

func (r *PostgresRepo) GetElementsByCategory(modelID, category string) ([]*model.Element, error) {
	rows, err := r.db.Query(`SELECT id, model_id, ifc_guid, name, type, category, parent_id, floor_name, aabb_min, aabb_max, properties, mesh_lods, geometry_hash FROM elements WHERE model_id = $1 AND category = $2`, modelID, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var elements []*model.Element
	for rows.Next() {
		e := &model.Element{}
		var aabbMin, aabbMax, props, lods string
		if err := rows.Scan(&e.ID, &e.ModelID, &e.IFCGUID, &e.Name, &e.Type, &e.Category, &e.ParentID, &e.FloorName, &aabbMin, &aabbMax, &props, &lods, &e.GeometryHash); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(props), &e.Properties)
		json.Unmarshal([]byte(lods), &e.MeshLODs)
		elements = append(elements, e)
	}
	return elements, nil
}

func (r *PostgresRepo) GetElementsByIDs(ids []string) ([]*model.Element, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := `SELECT id, model_id, ifc_guid, name, type, category, parent_id, floor_name, aabb_min, aabb_max, properties, mesh_lods, geometry_hash FROM elements WHERE id = ANY($1)`
	rows, err := r.db.Query(query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var elements []*model.Element
	for rows.Next() {
		e := &model.Element{}
		var aabbMin, aabbMax, props, lods string
		if err := rows.Scan(&e.ID, &e.ModelID, &e.IFCGUID, &e.Name, &e.Type, &e.Category, &e.ParentID, &e.FloorName, &aabbMin, &aabbMax, &props, &lods, &e.GeometryHash); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(props), &e.Properties)
		json.Unmarshal([]byte(lods), &e.MeshLODs)
		elements = append(elements, e)
	}
	return elements, nil
}

func (r *PostgresRepo) CreateSpatialNode(n *model.SpatialNode) error {
	_, err := r.db.Exec(
		`INSERT INTO spatial_nodes (id, model_id, parent_id, ifc_guid, name, type, level) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		n.ID, n.ModelID, n.ParentID, n.IFCGUID, n.Name, n.Type, n.Level,
	)
	return err
}

func (r *PostgresRepo) GetSpatialTree(modelID string) ([]*model.SpatialNode, error) {
	rows, err := r.db.Query(`SELECT id, model_id, parent_id, ifc_guid, name, type, level FROM spatial_nodes WHERE model_id = $1 ORDER BY level, name`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	nodeMap := make(map[string]*model.SpatialNode)
	var roots []*model.SpatialNode
	for rows.Next() {
		n := &model.SpatialNode{}
		if err := rows.Scan(&n.ID, &n.ModelID, &n.ParentID, &n.IFCGUID, &n.Name, &n.Type, &n.Level); err != nil {
			return nil, err
		}
		nodeMap[n.ID] = n
		if n.ParentID == nil {
			roots = append(roots, n)
		}
	}
	for _, n := range nodeMap {
		if n.ParentID != nil {
			if parent, ok := nodeMap[*n.ParentID]; ok {
				parent.Children = append(parent.Children, n)
			}
		}
	}
	return roots, nil
}

func (r *PostgresRepo) CreateMeshChunk(chunk *model.MeshChunk) error {
	_, err := r.db.Exec(
		`INSERT INTO mesh_chunks (id, model_id, lod, octree_node_id, data, vertex_count, index_count) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		chunk.ID, chunk.ModelID, chunk.LOD, chunk.OctreeNodeID, chunk.Data, chunk.VertexCount, chunk.IndexCount,
	)
	return err
}

func (r *PostgresRepo) GetMeshChunks(modelID string, lod int, nodeIDs []string) ([]*model.MeshChunk, error) {
	if len(nodeIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, model_id, lod, octree_node_id, data, vertex_count, index_count FROM mesh_chunks WHERE model_id = $1 AND lod = $2 AND octree_node_id = ANY($3)`
	rows, err := r.db.Query(query, modelID, lod, nodeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chunks []*model.MeshChunk
	for rows.Next() {
		c := &model.MeshChunk{}
		if err := rows.Scan(&c.ID, &c.ModelID, &c.LOD, &c.OctreeNodeID, &c.Data, &c.VertexCount, &c.IndexCount); err != nil {
			return nil, err
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

func (r *PostgresRepo) CreateCollisionTask(t *model.CollisionTask) error {
	groupA, _ := json.Marshal(t.GroupA)
	groupB, _ := json.Marshal(t.GroupB)
	_, err := r.db.Exec(
		`INSERT INTO collision_tasks (id, model_id, group_a, group_b, threshold, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		t.ID, t.ModelID, groupA, groupB, t.Threshold, t.Status,
	)
	return err
}

func (r *PostgresRepo) UpdateCollisionTaskStatus(id, status string) error {
	if status == "completed" {
		_, err := r.db.Exec(`UPDATE collision_tasks SET status = $2, completed_at = NOW() WHERE id = $1`, id, status)
		return err
	}
	_, err := r.db.Exec(`UPDATE collision_tasks SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *PostgresRepo) CreateCollisionResult(cr *model.CollisionResult) error {
	_, err := r.db.Exec(
		`INSERT INTO collision_results (id, task_id, element_a_id, element_a_name, element_a_type, element_a_floor, element_b_id, element_b_name, element_b_type, element_b_floor, collision_type, collision_point, penetration, severity)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		cr.ID, cr.TaskID, cr.ElementAID, cr.ElementAName, cr.ElementAType, cr.ElementAFloor,
		cr.ElementBID, cr.ElementBName, cr.ElementBType, cr.ElementBFloor,
		cr.CollisionType,
		fmt.Sprintf(`[%f,%f,%f]`, cr.CollisionPoint[0], cr.CollisionPoint[1], cr.CollisionPoint[2]),
		cr.Penetration, cr.Severity,
	)
	return err
}

func (r *PostgresRepo) GetCollisionResults(taskID string) ([]*model.CollisionResult, error) {
	rows, err := r.db.Query(
		`SELECT id, task_id, element_a_id, element_a_name, element_a_type, element_a_floor, element_b_id, element_b_name, element_b_type, element_b_floor, collision_type, collision_point, penetration, severity FROM collision_results WHERE task_id = $1 ORDER BY severity, penetration DESC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*model.CollisionResult
	for rows.Next() {
		cr := &model.CollisionResult{}
		var pointStr string
		if err := rows.Scan(&cr.ID, &cr.TaskID, &cr.ElementAID, &cr.ElementAName, &cr.ElementAType, &cr.ElementAFloor, &cr.ElementBID, &cr.ElementBName, &cr.ElementBType, &cr.ElementBFloor, &cr.CollisionType, &pointStr, &cr.Penetration, &cr.Severity); err != nil {
			return nil, err
		}
		var pt [3]float64
		json.Unmarshal([]byte(pointStr), &pt)
		cr.CollisionPoint = pt
		results = append(results, cr)
	}
	return results, nil
}

func (r *PostgresRepo) GetCollisionTask(id string) (*model.CollisionTask, error) {
	row := r.db.QueryRow(`SELECT id, model_id, group_a, group_b, threshold, status, created_at, completed_at FROM collision_tasks WHERE id = $1`, id)
	t := &model.CollisionTask{}
	var groupA, groupB string
	var completedAt sql.NullTime
	err := row.Scan(&t.ID, &t.ModelID, &groupA, &groupB, &t.Threshold, &t.Status, &t.CreatedAt, &completedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(groupA), &t.GroupA)
	json.Unmarshal([]byte(groupB), &t.GroupB)
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	return t, nil
}

func (r *PostgresRepo) scanElement(row *sql.Row) (*model.Element, error) {
	e := &model.Element{}
	var aabbMin, aabbMax, props, lods string
	err := row.Scan(&e.ID, &e.ModelID, &e.IFCGUID, &e.Name, &e.Type, &e.Category, &e.ParentID, &e.FloorName, &aabbMin, &aabbMax, &props, &lods, &e.GeometryHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(props), &e.Properties)
	json.Unmarshal([]byte(lods), &e.MeshLODs)
	return e, nil
}
