package model

import "time"

type ConstructionPlan struct {
	ID        string    `json:"id"`
	ModelID   string    `json:"modelId"`
	Name      string    `json:"name"`
	StartDate string    `json:"startDate"`
	EndDate   string    `json:"endDate"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Phases    []ConstructionPhase `json:"phases,omitempty"`
}

type ConstructionPhase struct {
	ID          string   `json:"id"`
	PlanID      string   `json:"planId"`
	Name        string   `json:"name"`
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	ElementIDs  []string `json:"elementIds"`
	Color       string   `json:"color,omitempty"`
	SortOrder   int      `json:"sortOrder"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CreateConstructionPlanRequest struct {
	ModelID   string `json:"modelId"`
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type UpdateConstructionPlanRequest struct {
	Name      *string `json:"name,omitempty"`
	StartDate *string `json:"startDate,omitempty"`
	EndDate   *string `json:"endDate,omitempty"`
}

type CreateConstructionPhaseRequest struct {
	Name       string   `json:"name"`
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	ElementIDs []string `json:"elementIds"`
}

type UpdateConstructionPhaseRequest struct {
	Name       *string  `json:"name,omitempty"`
	StartDate  *string  `json:"startDate,omitempty"`
	EndDate    *string  `json:"endDate,omitempty"`
	ElementIDs []string `json:"elementIds,omitempty"`
	SortOrder  *int     `json:"sortOrder,omitempty"`
}

type PhaseOverlapError struct {
	PhaseA string
	PhaseB string
	Msg    string
}

func (e *PhaseOverlapError) Error() string {
	return e.Msg
}
