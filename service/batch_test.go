package service

import (
	"context"
	"testing"

	"example.com/mestransform/domain"
)

func TestRunBatchUsesWorkersAndPersistsJob(t *testing.T) {
	svc, repo := testService(t)
	requests := []struct{ key string }{{"batch-a"}, {"batch-b"}, {"batch-c"}}
	input := make([]domainRequestAlias, 0, len(requests))
	for _, item := range requests {
		input = append(input, domainRequestAlias{request: testRequest(item.key)})
	}
	real := unwrapRequests(input)
	outcome, err := svc.RunBatch(context.Background(), "nightly", real, 2)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Job.Processed != 3 || len(outcome.Records) != 3 {
		t.Fatalf("outcome %#v", outcome)
	}
	stored, err := repo.BatchByID(outcome.Job.ID)
	if err != nil || stored.Status != "completed" {
		t.Fatalf("job %#v %v", stored, err)
	}
}

type domainRequestAlias struct{ request interfaceRequest }
type interfaceRequest = domain.ConversionRequest

func unwrapRequests(values []domainRequestAlias) []domain.ConversionRequest {
	result := make([]domain.ConversionRequest, 0, len(values))
	for _, value := range values {
		result = append(result, value.request)
	}
	return result
}
