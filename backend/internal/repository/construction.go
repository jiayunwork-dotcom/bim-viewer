package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"bim-viewer/internal/model"
)

func (r *PostgresRepo) MigrateConstruction() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS construction_plans (
			id VARCHAR(64) PRIMARY KEY,
			model_id VARCHAR(64) NOT NULL REFERENCES models(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS construction_phases (
			id VARCHAR(64) PRIMARY KEY,
			plan_id VARCHAR(64) NOT NULL REFERENCES construction_plans(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			start_date DATE NOT NULL,
			end_date DATE NOT NULL,
			element_ids JSONB DEFAULT '[]',
			color VARCHAR(16) DEFAULT '#409EFF',
			sort_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_construction_plans_model_id ON construction_plans(model_id)`,
		`CREATE INDEX IF NOT EXISTS idx_construction_phases_plan_id ON construction_phases(plan_id)`,
	}
	for _, m := range migrations {
		if _, err := r.db.Exec(m); err != nil {
			return fmt.Errorf("construction migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

func (r *PostgresRepo) CreateConstructionPlan(p *model.ConstructionPlan) error {
	_, err := r.db.Exec(
		`INSERT INTO construction_plans (id, model_id, name, start_date, end_date, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.ModelID, p.Name, p.StartDate, p.EndDate, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PostgresRepo) GetConstructionPlan(id string) (*model.ConstructionPlan, error) {
	row := r.db.QueryRow(
		`SELECT id, model_id, name, start_date, end_date, created_at, updated_at
		 FROM construction_plans WHERE id = $1`, id,
	)
	p := &model.ConstructionPlan{}
	err := row.Scan(&p.ID, &p.ModelID, &p.Name, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetConstructionPlan scan error: %w", err)
	}
	phases, err := r.GetConstructionPhasesByPlan(id)
	if err != nil {
		return nil, err
	}
	p.Phases = phases
	return p, nil
}

func (r *PostgresRepo) ListConstructionPlansByModel(modelID string) ([]*model.ConstructionPlan, error) {
	rows, err := r.db.Query(
		`SELECT id, model_id, name, start_date, end_date, created_at, updated_at
		 FROM construction_plans WHERE model_id = $1 ORDER BY created_at DESC`, modelID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plans []*model.ConstructionPlan
	for rows.Next() {
		p := &model.ConstructionPlan{}
		if err := rows.Scan(&p.ID, &p.ModelID, &p.Name, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *PostgresRepo) UpdateConstructionPlan(id string, req *model.UpdateConstructionPlanRequest) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", argIdx))
		d, err := model.ParseDateOnly(*req.StartDate)
		if err != nil {
			return fmt.Errorf("invalid startDate: %w", err)
		}
		args = append(args, d)
		argIdx++
	}
	if req.EndDate != nil {
		sets = append(sets, fmt.Sprintf("end_date = $%d", argIdx))
		d, err := model.ParseDateOnly(*req.EndDate)
		if err != nil {
			return fmt.Errorf("invalid endDate: %w", err)
		}
		args = append(args, d)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE construction_plans SET %s WHERE id = $%d", joinStrings(sets, ", "), argIdx)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PostgresRepo) DeleteConstructionPlan(id string) error {
	_, err := r.db.Exec(`DELETE FROM construction_plans WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) CreateConstructionPhase(ph *model.ConstructionPhase) error {
	elemIDsJSON, _ := json.Marshal(ph.ElementIDs)
	_, err := r.db.Exec(
		`INSERT INTO construction_phases (id, plan_id, name, start_date, end_date, element_ids, color, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		ph.ID, ph.PlanID, ph.Name, ph.StartDate, ph.EndDate, elemIDsJSON, ph.Color, ph.SortOrder, ph.CreatedAt, ph.UpdatedAt,
	)
	return err
}

func (r *PostgresRepo) GetConstructionPhase(id string) (*model.ConstructionPhase, error) {
	row := r.db.QueryRow(
		`SELECT id, plan_id, name, start_date, end_date, element_ids, color, sort_order, created_at, updated_at
		 FROM construction_phases WHERE id = $1`, id,
	)
	return r.scanPhase(row)
}

func (r *PostgresRepo) GetConstructionPhasesByPlan(planID string) ([]model.ConstructionPhase, error) {
	rows, err := r.db.Query(
		`SELECT id, plan_id, name, start_date, end_date, element_ids, color, sort_order, created_at, updated_at
		 FROM construction_phases WHERE plan_id = $1 ORDER BY sort_order, start_date`, planID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var phases []model.ConstructionPhase
	for rows.Next() {
		ph, err := r.scanPhaseFromRows(rows)
		if err != nil {
			return nil, err
		}
		phases = append(phases, *ph)
	}
	return phases, nil
}

func (r *PostgresRepo) UpdateConstructionPhase(id string, req *model.UpdateConstructionPhaseRequest) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.StartDate != nil {
		sets = append(sets, fmt.Sprintf("start_date = $%d", argIdx))
		d, err := model.ParseDateOnly(*req.StartDate)
		if err != nil {
			return fmt.Errorf("invalid startDate: %w", err)
		}
		args = append(args, d)
		argIdx++
	}
	if req.EndDate != nil {
		sets = append(sets, fmt.Sprintf("end_date = $%d", argIdx))
		d, err := model.ParseDateOnly(*req.EndDate)
		if err != nil {
			return fmt.Errorf("invalid endDate: %w", err)
		}
		args = append(args, d)
		argIdx++
	}
	if req.ElementIDs != nil {
		sets = append(sets, fmt.Sprintf("element_ids = $%d", argIdx))
		elemIDsJSON, _ := json.Marshal(req.ElementIDs)
		args = append(args, elemIDsJSON)
		argIdx++
	}
	if req.SortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE construction_phases SET %s WHERE id = $%d", joinStrings(sets, ", "), argIdx)
	_, err := r.db.Exec(query, args...)
	return err
}

func (r *PostgresRepo) DeleteConstructionPhase(id string) error {
	_, err := r.db.Exec(`DELETE FROM construction_phases WHERE id = $1`, id)
	return err
}

func (r *PostgresRepo) scanPhase(row *sql.Row) (*model.ConstructionPhase, error) {
	ph := &model.ConstructionPhase{}
	var elemIDsStr string
	err := row.Scan(&ph.ID, &ph.PlanID, &ph.Name, &ph.StartDate, &ph.EndDate, &elemIDsStr, &ph.Color, &ph.SortOrder, &ph.CreatedAt, &ph.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(elemIDsStr), &ph.ElementIDs)
	if ph.ElementIDs == nil {
		ph.ElementIDs = []string{}
	}
	return ph, nil
}

func (r *PostgresRepo) scanPhaseFromRows(rows *sql.Rows) (*model.ConstructionPhase, error) {
	ph := &model.ConstructionPhase{}
	var elemIDsStr string
	err := rows.Scan(&ph.ID, &ph.PlanID, &ph.Name, &ph.StartDate, &ph.EndDate, &elemIDsStr, &ph.Color, &ph.SortOrder, &ph.CreatedAt, &ph.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(elemIDsStr), &ph.ElementIDs)
	if ph.ElementIDs == nil {
		ph.ElementIDs = []string{}
	}
	return ph, nil
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, s := range items {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
