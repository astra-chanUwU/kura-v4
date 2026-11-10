package controllers

import (
	"strconv"

	"github.com/astra-chanUwU/kura-v4/internal/dto"
	"github.com/astra-chanUwU/kura-v4/internal/jobs"
	"github.com/astra-chanUwU/kura-v4/internal/model"
	"github.com/gofiber/fiber/v3"
)

// JobAdmin serves the moderator view of exhausted background jobs.
type JobAdmin struct{ jobs *jobs.Service }

func NewJobAdmin(svc *jobs.Service) *JobAdmin { return &JobAdmin{jobs: svc} }

func (h *JobAdmin) RegisterRoutes(api fiber.Router) {
	api.Get("/admin/jobs", h.list)
	api.Post("/admin/jobs/:id/retry", h.retry)
}

func (h *JobAdmin) list(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	limit, offset := queryPage(c)
	page, err := h.jobs.ListFailed(c.Context(), *user, c.Query("state"), limit, offset)
	if err != nil {
		return writeError(c, err)
	}
	out := make([]dto.FailedJob, 0, len(page.Jobs))
	for _, job := range page.Jobs {
		out = append(out, failedJobDTO(job))
	}
	return c.JSON(dto.FailedJobPage{Total: page.Total, Jobs: out})
}

func (h *JobAdmin) retry(c fiber.Ctx) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return fiber.NewError(400, "That job id is not valid.")
	}
	if err := h.jobs.Retry(c.Context(), *user, id); err != nil {
		return writeError(c, err)
	}
	return c.SendStatus(204)
}

func failedJobDTO(job model.FailedJob) dto.FailedJob {
	return dto.FailedJob{
		ID: job.ID, Kind: job.Kind, Attempts: job.Attempts, MaxAttempts: job.MaxAttempts,
		LastError: job.LastError, UpdatedAt: job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
