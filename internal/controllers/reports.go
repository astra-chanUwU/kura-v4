package controllers

import (
	"errors"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/astra-chanUwU/kura-v4/internal/reports"
	"github.com/gofiber/fiber/v3"
)

const reportTime = "2006-01-02T15:04:05Z07:00"

// Reports serves member reports and the moderator queue.
type Reports struct{ service *reports.Service }

func NewReports(s *reports.Service) *Reports { return &Reports{service: s} }

func (h *Reports) RegisterRoutes(api fiber.Router) {
	api.Post("/posts/:id/reports", h.create)
	api.Get("/admin/reports", h.list)
	api.Post("/admin/reports/:id/dismiss", h.dismiss)
}

func (h *Reports) create(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	var in dto.CreatePostReport
	if err := c.Bind().Body(&in); err != nil {
		return fiber.NewError(400, "Invalid request.")
	}
	report, err := h.service.Create(c.Context(), *user, c.Params("id"), in.Reason, in.Note)
	if err != nil {
		return writeReportError(c, err)
	}
	return c.Status(201).JSON(reportDTO(report))
}

func (h *Reports) list(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.service.ListOpen(c.Context(), *user, limit, offset)
	if err != nil {
		return writeReportError(c, err)
	}
	out := make([]dto.PostReport, 0, len(page.Reports))
	for _, report := range page.Reports {
		out = append(out, reportDTO(report))
	}
	return c.JSON(dto.PostReportPage{Total: page.Total, Reports: out})
}

func (h *Reports) dismiss(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	report, err := h.service.Dismiss(c.Context(), *user, c.Params("id"))
	if err != nil {
		return writeReportError(c, err)
	}
	return c.JSON(reportDTO(report))
}

func reportDTO(report model.PostReport) dto.PostReport {
	out := dto.PostReport{
		ID: report.ID, PostID: report.PostID, ThumbnailURL: "/media/posts/" + report.PostID + "/thumb",
		ReporterUsername: report.ReporterUsername, Reason: report.Reason, Note: report.Note,
		Status: report.Status, CreatedAt: report.CreatedAt.UTC().Format(reportTime),
	}
	if !report.ResolvedAt.IsZero() {
		out.ResolvedAt = report.ResolvedAt.UTC().Format(reportTime)
	}
	out.ResolverID = report.ResolverID
	return out
}

func writeReportError(c fiber.Ctx, err error) error {
	var vErr *reports.ValidationError
	switch {
	case errors.As(err, &vErr):
		return fiber.NewError(400, vErr.Message)
	case errors.Is(err, reports.ErrForbidden):
		return fiber.NewError(403, "You don't have permission to do that.")
	case errors.Is(err, reports.ErrNotFound):
		return fiber.NewError(404, "Not found.")
	default:
		return err
	}
}
