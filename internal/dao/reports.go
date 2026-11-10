package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/astra-chanUwU/kura-v4/internal/dao/sqlcgen"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/google/uuid"
)

type ReportsDAO struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewReports(conn *sql.DB) *ReportsDAO {
	return &ReportsDAO{db: conn, q: sqlcgen.New(conn)}
}

func (d *ReportsDAO) WithTx(tx *sql.Tx) *ReportsDAO {
	return &ReportsDAO{db: d.db, q: sqlcgen.New(tx)}
}

func (d *ReportsDAO) DB() *sql.DB { return d.db }

func reportFrom(id, postID, reporterID uuid.UUID, username, reason, note, status string, created time.Time, resolved sql.NullTime, resolver uuid.NullUUID) model.PostReport {
	out := model.PostReport{
		ID: id.String(), PostID: postID.String(), ReporterID: reporterID.String(),
		ReporterUsername: username, Reason: reason, Note: note, Status: status, CreatedAt: created,
	}
	if resolved.Valid {
		out.ResolvedAt = resolved.Time
	}
	if resolver.Valid {
		out.ResolverID = resolver.UUID.String()
	}
	return out
}

func (d *ReportsDAO) ReportPostAccess(ctx context.Context, postID string) (model.ReportPostAccess, error) {
	uid, err := uuid.Parse(postID)
	if err != nil {
		return model.ReportPostAccess{}, sql.ErrNoRows
	}
	r, err := d.q.ReportPostAccess(ctx, uid)
	if err != nil {
		return model.ReportPostAccess{}, err
	}
	access := model.ReportPostAccess{Rating: r.Rating, Hidden: r.Hidden, Deleted: r.Deleted}
	if r.OwnerID.Valid {
		access.OwnerID = r.OwnerID.UUID.String()
	}
	return access, nil
}

// GetOpenPostReport reports whether this member already has an open
// report on the post. A missing row is false, not an error.
func (d *ReportsDAO) GetOpenPostReport(ctx context.Context, postID, reporterID string) (bool, error) {
	post, err := uuid.Parse(postID)
	if err != nil {
		return false, nil
	}
	reporter, err := uuid.Parse(reporterID)
	if err != nil {
		return false, nil
	}
	_, err = d.q.GetOpenPostReport(ctx, sqlcgen.GetOpenPostReportParams{PostID: post, ReporterID: reporter})
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *ReportsDAO) CreatePostReport(ctx context.Context, postID, reporterID, reason, note string) (model.PostReport, error) {
	post, err := uuid.Parse(postID)
	if err != nil {
		return model.PostReport{}, err
	}
	reporter, err := uuid.Parse(reporterID)
	if err != nil {
		return model.PostReport{}, err
	}
	id, err := d.q.CreatePostReport(ctx, sqlcgen.CreatePostReportParams{
		PostID: post, ReporterID: reporter, Reason: reason, Note: note,
	})
	if err != nil {
		return model.PostReport{}, err
	}
	return d.GetPostReport(ctx, id.String())
}

func (d *ReportsDAO) GetPostReport(ctx context.Context, id string) (model.PostReport, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.PostReport{}, sql.ErrNoRows
	}
	r, err := d.q.GetPostReport(ctx, uid)
	if err != nil {
		return model.PostReport{}, err
	}
	return reportFrom(r.ID, r.PostID, r.ReporterID, r.ReporterUsername, r.Reason, r.Note, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID), nil
}

func (d *ReportsDAO) ListOpenPostReports(ctx context.Context, limit, offset int32) ([]model.PostReport, error) {
	rows, err := d.q.ListOpenPostReports(ctx, sqlcgen.ListOpenPostReportsParams{RowLimit: limit, RowOffset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]model.PostReport, 0, len(rows))
	for _, r := range rows {
		out = append(out, reportFrom(r.ID, r.PostID, r.ReporterID, r.ReporterUsername, r.Reason, r.Note, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID))
	}
	return out, nil
}

func (d *ReportsDAO) CountOpenPostReports(ctx context.Context) (int64, error) {
	return d.q.CountOpenPostReports(ctx)
}

func (d *ReportsDAO) DismissPostReport(ctx context.Context, id, resolverID string) (model.PostReport, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return model.PostReport{}, sql.ErrNoRows
	}
	resolver, err := uuid.Parse(resolverID)
	if err != nil {
		return model.PostReport{}, err
	}
	r, err := d.q.DismissPostReport(ctx, sqlcgen.DismissPostReportParams{
		ID: uid, ResolverID: uuid.NullUUID{UUID: resolver, Valid: true},
	})
	if err != nil {
		return model.PostReport{}, err
	}
	return reportFrom(r.ID, r.PostID, r.ReporterID, r.ReporterUsername, r.Reason, r.Note, r.Status, r.CreatedAt, r.ResolvedAt, r.ResolverID), nil
}
