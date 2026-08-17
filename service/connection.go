package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/mapping"
)

type ConnectionResult struct {
	ProfileName string              `json:"profile_name"`
	Vendor      domain.SourceVendor `json:"vendor"`
	Reachable   bool                `json:"reachable"`
	ReadOnly    bool                `json:"read_only"`
	Checks      []ConnectionCheck   `json:"checks"`
	Summary     string              `json:"summary"`
}

type ConnectionCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type ConnectionService struct{}

func NewConnectionService() *ConnectionService { return &ConnectionService{} }

func (s *ConnectionService) Test(ctx context.Context, profile domain.ConnectionProfile, catalog domain.SourceCatalog) (ConnectionResult, error) {
	if err := ctx.Err(); err != nil {
		return ConnectionResult{}, err
	}
	result := ConnectionResult{ProfileName: profile.Name, Vendor: profile.Vendor, ReadOnly: profile.ReadOnly}
	profileErr := profile.Validate()
	result.Checks = append(result.Checks, ConnectionCheck{Name: "profile", Passed: profileErr == nil, Message: errorText(profileErr, "connection profile is valid")})
	catalogErr := catalog.Validate()
	result.Checks = append(result.Checks, ConnectionCheck{Name: "catalog", Passed: catalogErr == nil, Message: errorText(catalogErr, "catalog structure is readable")})
	permissionPassed := profile.ReadOnly
	permissionMessage := "read-only access confirmed"
	if !permissionPassed {
		permissionMessage = "source connection should be read-only"
	}
	result.Checks = append(result.Checks, ConnectionCheck{Name: "permission", Passed: permissionPassed, Message: permissionMessage})
	requiredTables := map[string]bool{"work_orders": false, "equipment": false, "quality_inspections": false, "materials": false}
	for _, table := range catalog.Tables {
		name := strings.ToLower(table.Name)
		if _, ok := requiredTables[name]; ok {
			requiredTables[name] = true
		}
	}
	missing := []string{}
	for name, found := range requiredTables {
		if !found {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	tablesPassed := len(missing) == 0
	tablesMessage := "required MES structures detected"
	if !tablesPassed {
		tablesMessage = "missing structures: " + strings.Join(missing, ", ")
	}
	result.Checks = append(result.Checks, ConnectionCheck{Name: "mes_structures", Passed: tablesPassed, Message: tablesMessage})
	result.Reachable = true
	for _, check := range result.Checks {
		if !check.Passed {
			result.Reachable = false
		}
	}
	if result.Reachable {
		result.Summary = "connection can be used for deterministic schema collection"
	} else {
		result.Summary = "connection requires configuration changes"
	}
	return result, nil
}

func (s *ConnectionService) VerifyTarget(ctx context.Context, catalog domain.SourceCatalog, target string) (mapping.CompatibilityReport, error) {
	if err := ctx.Err(); err != nil {
		return mapping.CompatibilityReport{}, err
	}
	dialect, err := mapping.DialectFor(target)
	if err != nil {
		return mapping.CompatibilityReport{}, err
	}
	return mapping.AssessCompatibility(catalog, dialect)
}

func (s *ConnectionService) Checklist(result ConnectionResult) []string {
	lines := make([]string, 0, len(result.Checks))
	for _, check := range result.Checks {
		state := "PASS"
		if !check.Passed {
			state = "FAIL"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s", state, check.Name, check.Message))
	}
	return lines
}

func errorText(err error, success string) string {
	if err != nil {
		return err.Error()
	}
	return success
}
