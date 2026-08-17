package audit

import (
	"fmt"
	"sort"
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

type Report struct {
	TotalRecords   int            `json:"total_records"`
	Completed      int            `json:"completed"`
	Previewed      int            `json:"previewed"`
	ByRequester    map[string]int `json:"by_requester"`
	ByTarget       map[string]int `json:"by_target"`
	TotalTables    int            `json:"total_tables"`
	TotalColumns   int            `json:"total_columns"`
	TotalIndexes   int            `json:"total_indexes"`
	TotalDownloads int            `json:"total_downloads"`
	EntityCounts   map[string]int `json:"entity_counts"`
}

type Reporter struct{ repo *storage.Repository }

func NewReporter(repo *storage.Repository) *Reporter { return &Reporter{repo: repo} }

func (r *Reporter) Build() (Report, error) {
	records, err := r.repo.ListRecords()
	if err != nil {
		return Report{}, err
	}
	downloads, err := r.repo.ListDownloads("")
	if err != nil {
		return Report{}, err
	}
	entities, err := r.repo.EntityCounts()
	if err != nil {
		return Report{}, err
	}
	report := Report{TotalRecords: len(records), TotalDownloads: len(downloads), EntityCounts: entities, ByRequester: map[string]int{}, ByTarget: map[string]int{}}
	for _, record := range records {
		report.ByRequester[record.RequestedBy]++
		report.ByTarget[record.TargetName]++
		report.TotalTables += record.Summary.Tables
		report.TotalColumns += record.Summary.Columns
		report.TotalIndexes += record.Summary.Indexes
		switch record.Status {
		case "completed":
			report.Completed++
		case "preview":
			report.Previewed++
		}
	}
	return report, nil
}

func (r Report) RequesterRanking() []string {
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(r.ByRequester))
	for name, count := range r.ByRequester {
		pairs = append(pairs, pair{name: name, count: count})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].name < pairs[j].name
	})
	result := make([]string, 0, len(pairs))
	for _, item := range pairs {
		result = append(result, fmt.Sprintf("%s:%d", item.name, item.count))
	}
	return result
}

func (r Report) RenderText() string {
	lines := []string{
		fmt.Sprintf("records: %d", r.TotalRecords),
		fmt.Sprintf("completed: %d", r.Completed),
		fmt.Sprintf("previews: %d", r.Previewed),
		fmt.Sprintf("tables: %d", r.TotalTables),
		fmt.Sprintf("columns: %d", r.TotalColumns),
		fmt.Sprintf("indexes: %d", r.TotalIndexes),
		fmt.Sprintf("downloads: %d", r.TotalDownloads),
	}
	for _, requester := range r.RequesterRanking() {
		lines = append(lines, "requester: "+requester)
	}
	targets := make([]string, 0, len(r.ByTarget))
	for target := range r.ByTarget {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	for _, target := range targets {
		lines = append(lines, fmt.Sprintf("target: %s:%d", target, r.ByTarget[target]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (r Report) Healthy() bool {
	if r.TotalRecords == 0 {
		return true
	}
	if r.Completed+r.Previewed != r.TotalRecords {
		return false
	}
	for _, entity := range []string{"WorkOrder", "Equipment", "QualityInspection", "Material"} {
		if _, ok := r.EntityCounts[entity]; !ok {
			return false
		}
	}
	return true
}

func ReportRecord(record domain.ConversionRecord) string {
	return fmt.Sprintf("%s %s->%s %s tables=%d", record.ID, record.SourceName, record.TargetName, record.Status, record.Summary.Tables)
}
