package service

import (
	"context"
	"fmt"

	"example.com/mestransform/domain"
)

type PreviewService struct{ conversion *ConversionService }

func NewPreviewService(conversion *ConversionService) *PreviewService {
	return &PreviewService{conversion: conversion}
}

func (s *PreviewService) DDL(ctx context.Context, request domain.ConversionRequest) (string, error) {
	preview, err := s.conversion.Preview(ctx, request)
	if err != nil {
		return "", err
	}
	if len(preview.Statements) == 0 {
		return "", fmt.Errorf("preview contains no statements")
	}
	text := ""
	for _, statement := range preview.Statements {
		text += statement.SQL + "\n"
	}
	return text, nil
}

func (s *PreviewService) Confirm(ctx context.Context, request domain.ConversionRequest, confirmed bool) (domain.ConversionRecord, error) {
	if !confirmed {
		return domain.ConversionRecord{}, fmt.Errorf("execution was not confirmed")
	}
	return s.conversion.Convert(ctx, request)
}

func (s *PreviewService) Compare(ctx context.Context, request domain.ConversionRequest, expectedTables int) (bool, error) {
	preview, err := s.conversion.Preview(ctx, request)
	if err != nil {
		return false, err
	}
	return preview.Summary.Tables == expectedTables && len(preview.Statements) >= expectedTables, nil
}
