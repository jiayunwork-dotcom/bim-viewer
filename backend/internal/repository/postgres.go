package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

func (r *PostgresRepo) GetDB() *sql.DB {
	return r.db
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
			severity VARCHAR(16),
			status VARCHAR(32) DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS collision_result_history (
			id VARCHAR(64) PRIMARY KEY,
			result_id VARCHAR(64) REFERENCES collision_results(id) ON DELETE CASCADE,
			task_id VARCHAR(64) REFERENCES collision_tasks(id) ON DELETE CASCADE,
			old_status VARCHAR(32) DEFAULT 'pending',
			new_status VARCHAR(32) NOT NULL,
			remark TEXT NOT NULL,
			operator VARCHAR(128) DEFAULT 'system',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='collision_results' AND column_name='status') THEN
				ALTER TABLE collision_results ADD COLUMN status VARCHAR(32) DEFAULT 'pending';
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='collision_results' AND column_name='created_at') THEN
				ALTER TABLE collision_results ADD COLUMN created_at TIMESTAMP DEFAULT NOW();
			END IF;
		END $$`,
		`DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='collision_results' AND column_name='updated_at') THEN
				ALTER TABLE collision_results ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();
			END IF;
		END $$`,
		`UPDATE collision_results SET status = 'pending' WHERE status IS NULL OR status = ''`,
		`CREATE INDEX IF NOT EXISTS idx_elements_model_id ON elements(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_elements_category ON elements(category)`,
		`CREATE INDEX IF NOT EXISTS idx_elements_geometry_hash ON elements(geometry_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_mesh_chunks_model_lod ON mesh_chunks(model_id, lod)`,
		`CREATE INDEX IF NOT EXISTS idx_collision_results_task ON collision_results(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_collision_results_status ON collision_results(status)`,
		`CREATE INDEX IF NOT EXISTS idx_collision_history_result ON collision_result_history(result_id)`,
		`CREATE INDEX IF NOT EXISTS idx_collision_history_task ON collision_result_history(task_id)`,
		`CREATE TABLE IF NOT EXISTS model_versions (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) REFERENCES models(id) ON DELETE CASCADE,
			version_number VARCHAR(32) NOT NULL,
			description TEXT,
			element_snapshot JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_model_versions_model_id ON model_versions(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_model_versions_version ON model_versions(model_id, version_number)`,
		`CREATE TABLE IF NOT EXISTS version_annotations (
			id VARCHAR(64) PRIMARY KEY,
			base_version_id VARCHAR(64) REFERENCES model_versions(id) ON DELETE CASCADE,
			compare_version_id VARCHAR(64) REFERENCES model_versions(id) ON DELETE CASCADE,
			element_id VARCHAR(64) NOT NULL,
			content TEXT NOT NULL,
			author VARCHAR(128) NOT NULL DEFAULT 'anonymous',
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_version_annotations_versions ON version_annotations(base_version_id, compare_version_id)`,
		`CREATE INDEX IF NOT EXISTS idx_version_annotations_element ON version_annotations(element_id)`,
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
	if cr.Status == "" {
		cr.Status = model.CollisionStatusPending
	}
	_, err := r.db.Exec(
		`INSERT INTO collision_results (id, task_id, element_a_id, element_a_name, element_a_type, element_a_floor, element_b_id, element_b_name, element_b_type, element_b_floor, collision_type, collision_point, penetration, severity, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())`,
		cr.ID, cr.TaskID, cr.ElementAID, cr.ElementAName, cr.ElementAType, cr.ElementAFloor,
		cr.ElementBID, cr.ElementBName, cr.ElementBType, cr.ElementBFloor,
		cr.CollisionType,
		fmt.Sprintf(`[%f,%f,%f]`, cr.CollisionPoint[0], cr.CollisionPoint[1], cr.CollisionPoint[2]),
		cr.Penetration, cr.Severity, cr.Status,
	)
	return err
}

func (r *PostgresRepo) GetCollisionResults(taskID string) ([]*model.CollisionResult, error) {
	rows, err := r.db.Query(
		`SELECT id, task_id, element_a_id, element_a_name, element_a_type, element_a_floor, element_b_id, element_b_name, element_b_type, element_b_floor, collision_type, collision_point, penetration, severity, status, created_at, updated_at FROM collision_results WHERE task_id = $1 ORDER BY severity, penetration DESC`,
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
		var status string
		if err := rows.Scan(&cr.ID, &cr.TaskID, &cr.ElementAID, &cr.ElementAName, &cr.ElementAType, &cr.ElementAFloor, &cr.ElementBID, &cr.ElementBName, &cr.ElementBType, &cr.ElementBFloor, &cr.CollisionType, &pointStr, &cr.Penetration, &cr.Severity, &status, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		var pt [3]float64
		json.Unmarshal([]byte(pointStr), &pt)
		cr.CollisionPoint = pt
		cr.Status = model.CollisionStatus(status)
		if cr.Status == "" {
			cr.Status = model.CollisionStatusPending
		}
		results = append(results, cr)
	}
	return results, nil
}

func (r *PostgresRepo) GetCollisionResultsByModel(modelID string) ([]*model.CollisionResult, error) {
	rows, err := r.db.Query(
		`SELECT cr.id, cr.task_id, cr.element_a_id, cr.element_a_name, cr.element_a_type, cr.element_a_floor, cr.element_b_id, cr.element_b_name, cr.element_b_type, cr.element_b_floor, cr.collision_type, cr.collision_point, cr.penetration, cr.severity, cr.status, cr.created_at, cr.updated_at 
		 FROM collision_results cr 
		 INNER JOIN collision_tasks ct ON cr.task_id = ct.id 
		 WHERE ct.model_id = $1 
		 ORDER BY cr.created_at DESC, cr.severity, cr.penetration DESC`,
		modelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*model.CollisionResult
	for rows.Next() {
		cr := &model.CollisionResult{}
		var pointStr string
		var status string
		if err := rows.Scan(&cr.ID, &cr.TaskID, &cr.ElementAID, &cr.ElementAName, &cr.ElementAType, &cr.ElementAFloor, &cr.ElementBID, &cr.ElementBName, &cr.ElementBType, &cr.ElementBFloor, &cr.CollisionType, &pointStr, &cr.Penetration, &cr.Severity, &status, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			return nil, err
		}
		var pt [3]float64
		json.Unmarshal([]byte(pointStr), &pt)
		cr.CollisionPoint = pt
		cr.Status = model.CollisionStatus(status)
		if cr.Status == "" {
			cr.Status = model.CollisionStatusPending
		}
		results = append(results, cr)
	}
	return results, nil
}

func (r *PostgresRepo) GetCollisionStats(taskID string) (*model.CollisionStats, error) {
	row := r.db.QueryRow(
		`SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN status = 'confirmed' THEN 1 END) as confirmed,
			COUNT(CASE WHEN status = 'false_positive' THEN 1 END) as false_positive,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved
		 FROM collision_results WHERE task_id = $1`,
		taskID,
	)
	stats := &model.CollisionStats{}
	err := row.Scan(&stats.Total, &stats.Pending, &stats.Confirmed, &stats.False, &stats.Resolved)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *PostgresRepo) GetCollisionStatsByModel(modelID string) (*model.CollisionStats, error) {
	row := r.db.QueryRow(
		`SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN cr.status = 'pending' THEN 1 END) as pending,
			COUNT(CASE WHEN cr.status = 'confirmed' THEN 1 END) as confirmed,
			COUNT(CASE WHEN cr.status = 'false_positive' THEN 1 END) as false_positive,
			COUNT(CASE WHEN cr.status = 'resolved' THEN 1 END) as resolved
		 FROM collision_results cr
		 INNER JOIN collision_tasks ct ON cr.task_id = ct.id
		 WHERE ct.model_id = $1`,
		modelID,
	)
	stats := &model.CollisionStats{}
	err := row.Scan(&stats.Total, &stats.Pending, &stats.Confirmed, &stats.False, &stats.Resolved)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *PostgresRepo) GetCollisionResult(id string) (*model.CollisionResult, error) {
	row := r.db.QueryRow(
		`SELECT id, task_id, element_a_id, element_a_name, element_a_type, element_a_floor, element_b_id, element_b_name, element_b_type, element_b_floor, collision_type, collision_point, penetration, severity, status, created_at, updated_at 
		 FROM collision_results WHERE id = $1`,
		id,
	)
	cr := &model.CollisionResult{}
	var pointStr string
	var status string
	err := row.Scan(&cr.ID, &cr.TaskID, &cr.ElementAID, &cr.ElementAName, &cr.ElementAType, &cr.ElementAFloor, &cr.ElementBID, &cr.ElementBName, &cr.ElementBType, &cr.ElementBFloor, &cr.CollisionType, &pointStr, &cr.Penetration, &cr.Severity, &status, &cr.CreatedAt, &cr.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pt [3]float64
	json.Unmarshal([]byte(pointStr), &pt)
	cr.CollisionPoint = pt
	cr.Status = model.CollisionStatus(status)
	return cr, nil
}

func (r *PostgresRepo) CreateCollisionHistory(h *model.CollisionResultHistory) error {
	_, err := r.db.Exec(
		`INSERT INTO collision_result_history (id, result_id, task_id, old_status, new_status, remark, operator, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		h.ID, h.ResultID, h.TaskID, h.OldStatus, h.NewStatus, h.Remark, h.Operator,
	)
	return err
}

func (r *PostgresRepo) GetCollisionHistory(resultID string) ([]*model.CollisionResultHistory, error) {
	rows, err := r.db.Query(
		`SELECT id, result_id, task_id, old_status, new_status, remark, operator, created_at 
		 FROM collision_result_history WHERE result_id = $1 ORDER BY created_at DESC`,
		resultID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []*model.CollisionResultHistory
	for rows.Next() {
		h := &model.CollisionResultHistory{}
		var oldStatus, newStatus string
		if err := rows.Scan(&h.ID, &h.ResultID, &h.TaskID, &oldStatus, &newStatus, &h.Remark, &h.Operator, &h.CreatedAt); err != nil {
			return nil, err
		}
		h.OldStatus = model.CollisionStatus(oldStatus)
		h.NewStatus = model.CollisionStatus(newStatus)
		history = append(history, h)
	}
	return history, nil
}

func (r *PostgresRepo) UpdateCollisionResultStatus(resultID string, newStatus model.CollisionStatus, remark string, operator string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRow(`SELECT status, task_id FROM collision_results WHERE id = $1 FOR UPDATE`, resultID)
	var oldStatus string
	var taskID string
	if err := row.Scan(&oldStatus, &taskID); err != nil {
		return err
	}

	if oldStatus == string(newStatus) {
		return tx.Commit()
	}

	_, err = tx.Exec(
		`UPDATE collision_results SET status = $2, updated_at = NOW() WHERE id = $1`,
		resultID, string(newStatus),
	)
	if err != nil {
		return err
	}

	historyID := generateUUID()
	_, err = tx.Exec(
		`INSERT INTO collision_result_history (id, result_id, task_id, old_status, new_status, remark, operator, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		historyID, resultID, taskID, oldStatus, string(newStatus), remark, operator,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepo) BatchUpdateCollisionStatus(resultIDs []string, newStatus model.CollisionStatus, remark string, operator string) (int, error) {
	if len(resultIDs) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	placeholders := make([]string, len(resultIDs))
	args := make([]interface{}, len(resultIDs))
	for i, id := range resultIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT id, status, task_id FROM collision_results WHERE id IN (%s) FOR UPDATE`,
		strings.Join(placeholders, ","),
	)
	rows, err := tx.Query(query, args...)
	if err != nil {
		return 0, err
	}

	type resultInfo struct {
		ID        string
		OldStatus string
		TaskID    string
	}
	var results []resultInfo
	for rows.Next() {
		var ri resultInfo
		if err := rows.Scan(&ri.ID, &ri.OldStatus, &ri.TaskID); err != nil {
			rows.Close()
			return 0, err
		}
		results = append(results, ri)
	}
	rows.Close()

	updated := 0
	for _, ri := range results {
		if ri.OldStatus == string(newStatus) {
			continue
		}

		_, err = tx.Exec(
			`UPDATE collision_results SET status = $2, updated_at = NOW() WHERE id = $1`,
			ri.ID, string(newStatus),
		)
		if err != nil {
			return 0, err
		}

		historyID := generateUUID()
		_, err = tx.Exec(
			`INSERT INTO collision_result_history (id, result_id, task_id, old_status, new_status, remark, operator, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
			historyID, ri.ID, ri.TaskID, ri.OldStatus, string(newStatus), remark, operator,
		)
		if err != nil {
			return 0, err
		}
		updated++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return updated, nil
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

func generateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func (r *PostgresRepo) CreateVersion(v *model.ModelVersion) error {
	snapshotJSON, _ := json.Marshal(v.ElementSnapshot)
	_, err := r.db.Exec(
		`INSERT INTO model_versions (id, model_id, version_number, description, element_snapshot, created_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())`,
		v.ID, v.ModelID, v.VersionNumber, v.Description, snapshotJSON,
	)
	return err
}

func (r *PostgresRepo) GetVersion(id string) (*model.ModelVersion, error) {
	row := r.db.QueryRow(
		`SELECT id, model_id, version_number, description, element_snapshot, created_at 
		 FROM model_versions WHERE id = $1`,
		id,
	)
	return r.scanVersion(row)
}

func (r *PostgresRepo) ListVersions(modelID string) ([]*model.ModelVersion, error) {
	rows, err := r.db.Query(
		`SELECT id, model_id, version_number, description, element_snapshot, created_at 
		 FROM model_versions WHERE model_id = $1 ORDER BY created_at DESC`,
		modelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []*model.ModelVersion
	for rows.Next() {
		v := &model.ModelVersion{}
		var snapshotStr string
		err := rows.Scan(&v.ID, &v.ModelID, &v.VersionNumber, &v.Description, &snapshotStr, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(snapshotStr), &v.ElementSnapshot)
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *PostgresRepo) GetNextVersionNumber(modelID string) (string, error) {
	row := r.db.QueryRow(
		`SELECT COUNT(*) FROM model_versions WHERE model_id = $1`,
		modelID,
	)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d", count+1), nil
}

func (r *PostgresRepo) DeleteVersion(id string) error {
	_, err := r.db.Exec(`DELETE FROM model_versions WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) scanVersion(row *sql.Row) (*model.ModelVersion, error) {
	v := &model.ModelVersion{}
	var snapshotStr string
	err := row.Scan(&v.ID, &v.ModelID, &v.VersionNumber, &v.Description, &snapshotStr, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(snapshotStr), &v.ElementSnapshot)
	return v, nil
}

func (r *PostgresRepo) CreateVersionAnnotation(a *model.VersionAnnotation) error {
	_, err := r.db.Exec(
		`INSERT INTO version_annotations (id, base_version_id, compare_version_id, element_id, content, author, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		a.ID, a.BaseVersionID, a.CompareVersionID, a.ElementID, a.Content, a.Author,
	)
	return err
}

func (r *PostgresRepo) GetVersionAnnotation(id string) (*model.VersionAnnotation, error) {
	row := r.db.QueryRow(
		`SELECT id, base_version_id, compare_version_id, element_id, content, author, created_at
		 FROM version_annotations WHERE id = $1`, id,
	)
	a := &model.VersionAnnotation{}
	err := row.Scan(&a.ID, &a.BaseVersionID, &a.CompareVersionID, &a.ElementID, &a.Content, &a.Author, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (r *PostgresRepo) ListVersionAnnotations(baseVersionID, compareVersionID string) ([]*model.VersionAnnotation, error) {
	rows, err := r.db.Query(
		`SELECT id, base_version_id, compare_version_id, element_id, content, author, created_at
		 FROM version_annotations 
		 WHERE base_version_id = $1 AND compare_version_id = $2
		 ORDER BY created_at DESC`,
		baseVersionID, compareVersionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var annotations []*model.VersionAnnotation
	for rows.Next() {
		a := &model.VersionAnnotation{}
		if err := rows.Scan(&a.ID, &a.BaseVersionID, &a.CompareVersionID, &a.ElementID, &a.Content, &a.Author, &a.CreatedAt); err != nil {
			return nil, err
		}
		annotations = append(annotations, a)
	}
	return annotations, nil
}

func (r *PostgresRepo) DeleteVersionAnnotation(id string) error {
	_, err := r.db.Exec(`DELETE FROM version_annotations WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) GetVersionAnnotationAuthor(id string) (string, error) {
	row := r.db.QueryRow(`SELECT author FROM version_annotations WHERE id = $1`, id)
	var author string
	err := row.Scan(&author)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return author, err
}
