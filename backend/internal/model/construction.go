package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type DateOnly struct {
	time.Time
}

const dateLayout = "2006-01-02"

func (d *DateOnly) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("invalid date format %q, expected YYYY-MM-DD: %w", s, err)
	}
	d.Time = t
	return nil
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, d.Time.Format(dateLayout))), nil
}

func (d DateOnly) String() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(dateLayout)
}

func (d DateOnly) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time.Format(dateLayout), nil
}

func (d *DateOnly) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		d.Time = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
		return nil
	case string:
		if v == "" {
			d.Time = time.Time{}
			return nil
		}
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			return fmt.Errorf("DateOnly.Scan: cannot parse string %q: %w", v, err)
		}
		d.Time = t
		return nil
	case []byte:
		s := string(v)
		if s == "" {
			d.Time = time.Time{}
			return nil
		}
		t, err := time.Parse(dateLayout, s)
		if err != nil {
			return fmt.Errorf("DateOnly.Scan: cannot parse bytes %q: %w", s, err)
		}
		d.Time = t
		return nil
	default:
		return fmt.Errorf("DateOnly.Scan: unsupported type %T", value)
	}
}

func ParseDateOnly(s string) (DateOnly, error) {
	if s == "" {
		return DateOnly{}, nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return DateOnly{}, err
	}
	return DateOnly{Time: t}, nil
}

func DateOnlyFromTime(t time.Time) DateOnly {
	return DateOnly{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)}
}

type ConstructionPlan struct {
	ID        string              `json:"id"`
	ModelID   string              `json:"modelId"`
	Name      string              `json:"name"`
	StartDate DateOnly            `json:"startDate"`
	EndDate   DateOnly            `json:"endDate"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Phases    []ConstructionPhase `json:"phases,omitempty"`
}

type ConstructionPhase struct {
	ID         string   `json:"id"`
	PlanID     string   `json:"planId"`
	Name       string   `json:"name"`
	StartDate  DateOnly `json:"startDate"`
	EndDate    DateOnly `json:"endDate"`
	ElementIDs []string `json:"elementIds"`
	Color      string   `json:"color,omitempty"`
	SortOrder  int      `json:"sortOrder"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
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
