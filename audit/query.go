package audit

import (
	"strings"

	"example.com/mestransform/domain"
	"example.com/mestransform/storage"
)

type Query struct{ repo *storage.Repository }

func NewQuery(repo *storage.Repository) *Query { return &Query{repo: repo} }

func (q *Query) RecordsByRequester(requester string) ([]domain.ConversionRecord, error) {
	records, err := q.repo.ListRecords()
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.ConversionRecord, 0, len(records))
	for _, record := range records {
		if strings.EqualFold(record.RequestedBy, requester) {
			filtered = append(filtered, record)
		}
	}
	return filtered, nil
}

func (q *Query) RecordsByStatus(status string) ([]domain.ConversionRecord, error) {
	records, err := q.repo.ListRecords()
	if err != nil {
		return nil, err
	}
	result := []domain.ConversionRecord{}
	for _, record := range records {
		if strings.EqualFold(record.Status, status) {
			result = append(result, record)
		}
	}
	return result, nil
}

func (q *Query) CountByTarget(target string) (int, error) {
	records, err := q.repo.ListRecords()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		if strings.EqualFold(record.TargetName, target) {
			count++
		}
	}
	return count, nil
}

func (q *Query) EntitySummary() (map[string]int, error) { return q.repo.EntityCounts() }
