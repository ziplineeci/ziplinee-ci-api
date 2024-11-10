package catalog

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/ziplineeci/ziplinee-ci-api/pkg/api"
	contracts "github.com/ziplineeci/ziplinee-ci-contracts"
)

// NewTracingService returns a new instance of a tracing Service.
func NewTracingService(s Service) Service {
	return &tracingService{s, "ziplinee"}
}

type tracingService struct {
	Service Service
	prefix  string
}

func (s *tracingService) CreateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (insertedCatalogEntity *contracts.CatalogEntity, err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "CreateCatalogEntity"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.CreateCatalogEntity(ctx, catalogEntity)
}

func (s *tracingService) UpdateCatalogEntity(ctx context.Context, catalogEntity contracts.CatalogEntity) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "UpdateCatalogEntity"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.UpdateCatalogEntity(ctx, catalogEntity)
}

func (s *tracingService) DeleteCatalogEntity(ctx context.Context, id string) (err error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, api.GetSpanName(s.prefix, "DeleteCatalogEntity"))
	defer func() { api.FinishSpanWithError(span, err) }()

	return s.Service.DeleteCatalogEntity(ctx, id)
}
