package service

import (
	"crypto/rand"
	"fmt"
	"time"
	"bim-viewer/internal/model"
	"bim-viewer/internal/repository"
)

type ConstructionService struct {
	repo *repository.PostgresRepo
}

func NewConstructionService(repo *repository.PostgresRepo) *ConstructionService {
	return &ConstructionService{repo: repo}
}

func (s *ConstructionService) CreatePlan(req *model.CreateConstructionPlanRequest) (*model.ConstructionPlan, error) {
	if req.ModelID == "" || req.Name == "" {
		return nil, fmt.Errorf("modelId and name are required")
	}
	if req.StartDate == "" || req.EndDate == "" {
		return nil, fmt.Errorf("startDate and endDate are required")
	}

	startDate, err := model.ParseDateOnly(req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate: %w", err)
	}
	endDate, err := model.ParseDateOnly(req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate: %w", err)
	}
	if startDate.After(endDate.Time) {
		return nil, fmt.Errorf("startDate must be before endDate")
	}

	now := time.Now().UTC()
	p := &model.ConstructionPlan{
		ID:        generateConstructionUUID(),
		ModelID:   req.ModelID,
		Name:      req.Name,
		StartDate: startDate,
		EndDate:   endDate,
		CreatedAt: now,
		UpdatedAt: now,
		Phases:    []model.ConstructionPhase{},
	}

	if err := s.repo.CreateConstructionPlan(p); err != nil {
		return nil, fmt.Errorf("failed to create construction plan: %w", err)
	}

	return p, nil
}

func (s *ConstructionService) GetPlan(id string) (*model.ConstructionPlan, error) {
	return s.repo.GetConstructionPlan(id)
}

func (s *ConstructionService) ListPlansByModel(modelID string) ([]*model.ConstructionPlan, error) {
	return s.repo.ListConstructionPlansByModel(modelID)
}

func (s *ConstructionService) UpdatePlan(id string, req *model.UpdateConstructionPlanRequest) (*model.ConstructionPlan, error) {
	existing, err := s.repo.GetConstructionPlan(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	if req.StartDate != nil || req.EndDate != nil {
		newStart := existing.StartDate
		newEnd := existing.EndDate
		if req.StartDate != nil {
			d, err := model.ParseDateOnly(*req.StartDate)
			if err != nil {
				return nil, fmt.Errorf("invalid startDate: %w", err)
			}
			newStart = d
		}
		if req.EndDate != nil {
			d, err := model.ParseDateOnly(*req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid endDate: %w", err)
			}
			newEnd = d
		}
		if newStart.After(newEnd.Time) {
			return nil, fmt.Errorf("startDate must be before endDate")
		}
	}

	if err := s.repo.UpdateConstructionPlan(id, req); err != nil {
		return nil, err
	}

	return s.repo.GetConstructionPlan(id)
}

func (s *ConstructionService) DeletePlan(id string) error {
	return s.repo.DeleteConstructionPlan(id)
}

func (s *ConstructionService) CreatePhase(planID string, req *model.CreateConstructionPhaseRequest) (*model.ConstructionPhase, error) {
	plan, err := s.repo.GetConstructionPlan(planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("construction plan not found")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("phase name is required")
	}
	if req.StartDate == "" || req.EndDate == "" {
		return nil, fmt.Errorf("phase startDate and endDate are required")
	}

	startDate, err := model.ParseDateOnly(req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate: %w", err)
	}
	endDate, err := model.ParseDateOnly(req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate: %w", err)
	}
	if startDate.After(endDate.Time) {
		return nil, fmt.Errorf("phase startDate must be before endDate")
	}
	if startDate.Before(plan.StartDate.Time) && !startDate.Equal(plan.StartDate.Time) {
		return nil, fmt.Errorf("phase startDate must not be earlier than plan startDate (%s)", plan.StartDate.String())
	}
	if endDate.After(plan.EndDate.Time) && !endDate.Equal(plan.EndDate.Time) {
		return nil, fmt.Errorf("phase endDate must not be later than plan endDate (%s)", plan.EndDate.String())
	}

	if err := s.validatePhaseOverlap(planID, "", startDate, endDate); err != nil {
		return nil, err
	}

	if err := s.validateElementUniqueness(planID, "", req.ElementIDs); err != nil {
		return nil, err
	}

	maxOrder := 0
	for _, ph := range plan.Phases {
		if ph.SortOrder > maxOrder {
			maxOrder = ph.SortOrder
		}
	}

	color := s.assignPhaseColor(len(plan.Phases))
	now := time.Now().UTC()

	ph := &model.ConstructionPhase{
		ID:             generateConstructionUUID(),
		PlanID:         planID,
		Name:           req.Name,
		StartDate:      startDate,
		EndDate:        endDate,
		ElementIDs:     req.ElementIDs,
		Color:          color,
		SortOrder:      maxOrder + 1,
		PredecessorIDs: []string{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if ph.ElementIDs == nil {
		ph.ElementIDs = []string{}
	}

	if err := s.repo.CreateConstructionPhase(ph); err != nil {
		return nil, fmt.Errorf("failed to create construction phase: %w", err)
	}

	return ph, nil
}

func (s *ConstructionService) UpdatePhase(phaseID string, req *model.UpdateConstructionPhaseRequest) (*model.ConstructionPhase, error) {
	existing, err := s.repo.GetConstructionPhase(phaseID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	newStart := existing.StartDate
	newEnd := existing.EndDate

	if req.StartDate != nil || req.EndDate != nil {
		if req.StartDate != nil {
			d, err := model.ParseDateOnly(*req.StartDate)
			if err != nil {
				return nil, fmt.Errorf("invalid startDate: %w", err)
			}
			newStart = d
		}
		if req.EndDate != nil {
			d, err := model.ParseDateOnly(*req.EndDate)
			if err != nil {
				return nil, fmt.Errorf("invalid endDate: %w", err)
			}
			newEnd = d
		}
		if newStart.After(newEnd.Time) {
			return nil, fmt.Errorf("phase startDate must be before endDate")
		}

		plan, err := s.repo.GetConstructionPlan(existing.PlanID)
		if err != nil {
			return nil, err
		}
		if plan != nil {
			if newStart.Before(plan.StartDate.Time) && !newStart.Equal(plan.StartDate.Time) {
				return nil, fmt.Errorf("phase startDate must not be earlier than plan startDate (%s)", plan.StartDate.String())
			}
			if newEnd.After(plan.EndDate.Time) && !newEnd.Equal(plan.EndDate.Time) {
				return nil, fmt.Errorf("phase endDate must not be later than plan endDate (%s)", plan.EndDate.String())
			}
		}

		if err := s.validatePhaseOverlap(existing.PlanID, phaseID, newStart, newEnd); err != nil {
			return nil, err
		}
	}

	if req.ElementIDs != nil {
		if err := s.validateElementUniqueness(existing.PlanID, phaseID, req.ElementIDs); err != nil {
			return nil, err
		}
	}

	if req.PredecessorIDs != nil {
		if err := s.validatePredecessors(phaseID, existing.PlanID, req.PredecessorIDs, newStart); err != nil {
			return nil, err
		}
	}

	if err := s.repo.UpdateConstructionPhase(phaseID, req); err != nil {
		return nil, err
	}

	return s.repo.GetConstructionPhase(phaseID)
}

func (s *ConstructionService) DeletePhase(phaseID string) error {
	return s.repo.DeleteConstructionPhase(phaseID)
}

func (s *ConstructionService) GetPhasesByPlan(planID string) ([]model.ConstructionPhase, error) {
	return s.repo.GetConstructionPhasesByPlan(planID)
}

func (s *ConstructionService) validatePhaseOverlap(planID, excludePhaseID string, startDate, endDate model.DateOnly) error {
	phases, err := s.repo.GetConstructionPhasesByPlan(planID)
	if err != nil {
		return err
	}

	for _, ph := range phases {
		if ph.ID == excludePhaseID {
			continue
		}
		overlap := startDate.Before(ph.EndDate.Time) && endDate.After(ph.StartDate.Time) &&
			!startDate.Equal(ph.EndDate.Time) && !endDate.Equal(ph.StartDate.Time)
		if overlap {
			return fmt.Errorf("phase time range overlaps with phase '%s' (%s ~ %s)", ph.Name, ph.StartDate.String(), ph.EndDate.String())
		}
	}
	return nil
}

func (s *ConstructionService) validateElementUniqueness(planID, excludePhaseID string, elementIDs []string) error {
	if len(elementIDs) == 0 {
		return nil
	}

	phases, err := s.repo.GetConstructionPhasesByPlan(planID)
	if err != nil {
		return err
	}

	elementPhaseMap := make(map[string]string)
	for _, ph := range phases {
		if ph.ID == excludePhaseID {
			continue
		}
		for _, eid := range ph.ElementIDs {
			elementPhaseMap[eid] = ph.Name
		}
	}

	for _, eid := range elementIDs {
		if phaseName, exists := elementPhaseMap[eid]; exists {
			return fmt.Errorf("element %s is already assigned to phase '%s'; each element can only belong to one phase", eid, phaseName)
		}
	}
	return nil
}

func (s *ConstructionService) assignPhaseColor(index int) string {
	colors := []string{
		"#409EFF", "#67C23A", "#E6A23C", "#F56C6C",
		"#909399", "#00BCD4", "#9C27B0", "#FF9800",
		"#795548", "#607D8B", "#3F51B5", "#009688",
	}
	return colors[index%len(colors)]
}

func (s *ConstructionService) validatePredecessors(phaseID, planID string, predecessorIDs []string, phaseStart model.DateOnly) error {
	phases, err := s.repo.GetConstructionPhasesByPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to get phases: %w", err)
	}

	phaseMap := make(map[string]*model.ConstructionPhase)
	for i := range phases {
		phaseMap[phases[i].ID] = &phases[i]
	}

	for _, predID := range predecessorIDs {
		if predID == phaseID {
			return fmt.Errorf("a phase cannot depend on itself")
		}
		pred, exists := phaseMap[predID]
		if !exists {
			return fmt.Errorf("predecessor phase '%s' does not exist", predID)
		}
		if phaseStart.Before(pred.EndDate.Time) && !phaseStart.Equal(pred.EndDate.Time) {
			return fmt.Errorf("phase start date (%s) must not be earlier than predecessor '%s' end date (%s)",
				phaseStart.String(), pred.Name, pred.EndDate.String())
		}
	}

	if err := s.validateNoCycle(phaseID, predecessorIDs, phaseMap); err != nil {
		return err
	}

	return nil
}

func (s *ConstructionService) validateNoCycle(phaseID string, newPredecessors []string, phaseMap map[string]*model.ConstructionPhase) error {
	adj := make(map[string][]string)
	for id, phase := range phaseMap {
		if id == phaseID {
			adj[id] = newPredecessors
		} else {
			adj[id] = phase.PredecessorIDs
		}
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(id string) bool {
		if recStack[id] {
			return true
		}
		if visited[id] {
			return false
		}
		visited[id] = true
		recStack[id] = true
		for _, pred := range adj[id] {
			if hasCycle(pred) {
				return true
			}
		}
		recStack[id] = false
		return false
	}

	for id := range phaseMap {
		if hasCycle(id) {
			return fmt.Errorf("dependency cycle detected involving phase '%s'", phaseMap[id].Name)
		}
	}

	return nil
}

func (s *ConstructionService) CalculateCriticalPath(planID string) (*model.CriticalPathResponse, error) {
	phases, err := s.repo.GetConstructionPhasesByPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get phases: %w", err)
	}
	if len(phases) == 0 {
		return &model.CriticalPathResponse{
			PhaseIDs:       []string{},
			TotalDuration:  0,
			PhaseDurations: make(map[string]int),
		}, nil
	}

	phaseMap := make(map[string]*model.ConstructionPhase)
	phaseDurations := make(map[string]int)
	for i := range phases {
		phaseMap[phases[i].ID] = &phases[i]
		duration := int(phases[i].EndDate.Time.Sub(phases[i].StartDate.Time).Hours()/24) + 1
		phaseDurations[phases[i].ID] = duration
	}

	dist := make(map[string]int)
	prev := make(map[string]string)
	for id := range phaseMap {
		dist[id] = phaseDurations[id]
		prev[id] = ""
	}

	topoOrder, err := s.topologicalSort(phaseMap)
	if err != nil {
		return nil, err
	}

	for _, id := range topoOrder {
		phase := phaseMap[id]
		for _, predID := range phase.PredecessorIDs {
			if dist[predID]+phaseDurations[id] > dist[id] {
				dist[id] = dist[predID] + phaseDurations[id]
				prev[id] = predID
			}
		}
	}

	maxDist := 0
	endPhaseID := ""
	for id, d := range dist {
		if d > maxDist {
			maxDist = d
			endPhaseID = id
		}
	}

	path := []string{}
	current := endPhaseID
	for current != "" {
		path = append([]string{current}, path...)
		current = prev[current]
	}

	return &model.CriticalPathResponse{
		PhaseIDs:       path,
		TotalDuration:  maxDist,
		PhaseDurations: phaseDurations,
	}, nil
}

func (s *ConstructionService) topologicalSort(phaseMap map[string]*model.ConstructionPhase) ([]string, error) {
	inDegree := make(map[string]int)
	for id := range phaseMap {
		inDegree[id] = 0
	}
	for _, phase := range phaseMap {
		for range phase.PredecessorIDs {
			inDegree[phase.ID]++
		}
	}

	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := []string{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)

		for _, phase := range phaseMap {
			for _, predID := range phase.PredecessorIDs {
				if predID == id {
					inDegree[phase.ID]--
					if inDegree[phase.ID] == 0 {
						queue = append(queue, phase.ID)
					}
				}
			}
		}
	}

	if len(result) != len(phaseMap) {
		return nil, fmt.Errorf("cycle detected in phase dependencies")
	}

	return result, nil
}

func generateConstructionUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("cp-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("cp-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
