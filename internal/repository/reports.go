package repository

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/dao"
	"github.com/astra-chanUwU/kura-v4/internal/model"
)

type Reports struct {
	reports *dao.ReportsDAO
}

func NewReports(d *dao.ReportsDAO) *Reports { return &Reports{reports: d} }

func (r *Reports) ReportPostAccess(ctx context.Context, postID string) (model.ReportPostAccess, error) {
	access, err := r.reports.ReportPostAccess(ctx, postID)
	return access, NotFound(err)
}

func (r *Reports) GetOpenPostReport(ctx context.Context, postID, reporterID string) (bool, error) {
	return r.reports.GetOpenPostReport(ctx, postID, reporterID)
}

func (r *Reports) CreatePostReport(ctx context.Context, postID, reporterID, reason, note string) (model.PostReport, error) {
	return r.reports.CreatePostReport(ctx, postID, reporterID, reason, note)
}

func (r *Reports) ListOpenPostReports(ctx context.Context, limit, offset int32) ([]model.PostReport, error) {
	return r.reports.ListOpenPostReports(ctx, limit, offset)
}

func (r *Reports) CountOpenPostReports(ctx context.Context) (int64, error) {
	return r.reports.CountOpenPostReports(ctx)
}

func (r *Reports) DismissPostReport(ctx context.Context, id, resolverID string) (model.PostReport, error) {
	report, err := r.reports.DismissPostReport(ctx, id, resolverID)
	return report, NotFound(err)
}
