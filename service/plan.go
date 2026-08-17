package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
)

type PlanStatus string

const (
	PlanDraft    PlanStatus = "draft"
	PlanApproved PlanStatus = "approved"
	PlanBlocked  PlanStatus = "blocked"
)

type MigrationStep struct {
	Number     int      `json:"number"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Tables     []string `json:"tables"`
	Statements []string `json:"statements"`
	Required   bool     `json:"required"`
	Completed  bool     `json:"completed"`
}

type MigrationPlan struct {
	ID            string          `json:"id"`
	Source        string          `json:"source"`
	Target        string          `json:"target"`
	Status        PlanStatus      `json:"status"`
	Steps         []MigrationStep `json:"steps"`
	Warnings      []string        `json:"warnings"`
	Approver      string          `json:"approver"`
	Compatibility string          `json:"compatibility"`
}

func BuildPlan(ctx context.Context, request domain.ConversionRequest) (MigrationPlan, error) {
	if err := ctx.Err(); err != nil {
		return MigrationPlan{}, err
	}
	if err := request.Validate(); err != nil {
		return MigrationPlan{}, err
	}
	dialect, err := mapping.DialectFor(request.TargetName)
	if err != nil {
		return MigrationPlan{}, err
	}
	mapper := mapping.NewMapper(dialect)
	preview, err := mapper.Preview(request.Catalog)
	if err != nil {
		return MigrationPlan{}, err
	}
	compatibility, err := mapping.AssessCompatibility(request.Catalog, dialect)
	if err != nil {
		return MigrationPlan{}, err
	}
	plan := MigrationPlan{ID: request.IdempotencyKey + "-plan", Source: request.SourceName, Target: request.TargetName, Status: PlanDraft, Warnings: append([]string(nil), preview.Warnings...), Compatibility: compatibility.SummaryText()}
	tables := request.Catalog.SortedTables()
	for index, table := range tables {
		step := MigrationStep{Number: index + 1, Kind: "table", Title: "Create " + table.Name, Tables: []string{table.Name}, Required: true}
		for _, statement := range preview.Statements {
			if strings.EqualFold(statement.Table, table.Name) {
				step.Statements = append(step.Statements, statement.SQL)
			}
		}
		plan.Steps = append(plan.Steps, step)
	}
	indexStatements := mapper.IndexStatements(request.Catalog)
	if len(indexStatements) > 0 {
		step := MigrationStep{Number: len(plan.Steps) + 1, Kind: "index", Title: "Create indexes", Required: false}
		for _, statement := range indexStatements {
			step.Statements = append(step.Statements, statement.SQL)
		}
		plan.Steps = append(plan.Steps, step)
	}
	if !compatibility.Ready {
		plan.Status = PlanBlocked
		plan.Warnings = append(plan.Warnings, "manual mappings must be resolved before approval")
	}
	return plan, nil
}

func (p MigrationPlan) Approve(actor string) (MigrationPlan, error) {
	if p.Status == PlanBlocked {
		return MigrationPlan{}, fmt.Errorf("plan has blocking mappings")
	}
	if strings.TrimSpace(actor) == "" {
		return MigrationPlan{}, fmt.Errorf("approver is required")
	}
	p.Status = PlanApproved
	p.Approver = actor
	return p, nil
}

func (p MigrationPlan) CompleteStep(number int) (MigrationPlan, error) {
	if p.Status != PlanApproved {
		return MigrationPlan{}, fmt.Errorf("plan must be approved")
	}
	found := false
	for index := range p.Steps {
		if p.Steps[index].Number == number {
			if p.Steps[index].Completed {
				return p, nil
			}
			p.Steps[index].Completed = true
			found = true
		}
	}
	if !found {
		return MigrationPlan{}, fmt.Errorf("step %d not found", number)
	}
	return p, nil
}

func (p MigrationPlan) Complete() bool {
	for _, step := range p.Steps {
		if step.Required && !step.Completed {
			return false
		}
	}
	return p.Status == PlanApproved
}

func (p MigrationPlan) OrderedSteps() []MigrationStep {
	result := append([]MigrationStep(nil), p.Steps...)
	sort.Slice(result, func(i, j int) bool { return result[i].Number < result[j].Number })
	return result
}

func (p MigrationPlan) Summary() string {
	completed := 0
	for _, step := range p.Steps {
		if step.Completed {
			completed++
		}
	}
	return fmt.Sprintf("%s %s->%s %d/%d steps status=%s", p.ID, p.Source, p.Target, completed, len(p.Steps), p.Status)
}
