package agentcli

import (
	"context"

	"github.com/astra-chanUwU/kura-v4/internal/model"
)

const reportsUsage = "usage: kura agent reports <create|list|dismiss>"

func (e *Engine) reports(ctx context.Context, args []string) int {
	if e.deps.Reports == nil {
		return e.fail("reports", "internal", "Reports are not available.")
	}
	if len(args) == 0 {
		return e.badUsage("reports", reportsUsage)
	}
	switch args[0] {
	case "create":
		return e.reportsCreate(ctx, args[1:])
	case "list":
		return e.reportsList(ctx, args[1:])
	case "dismiss":
		return e.reportsDismiss(ctx, args[1:])
	default:
		return e.badUsage("reports", reportsUsage)
	}
}

func (e *Engine) reportsCreate(ctx context.Context, args []string) int {
	const cmd = "reports.create"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "member") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("reason", "note"); err != nil || len(parsed.Pos) != 1 || !parsed.Has("reason") {
		return e.badUsage(cmd, "usage: kura agent reports create <post-id> --reason <illegal|underage|spam|wrong_rating|other> [--note text]")
	}
	report, err := e.deps.Reports.Create(ctx, user, parsed.Pos[0], parsed.Get("reason"), parsed.Get("note"))
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, reportView(report))
}

func (e *Engine) reportsList(ctx context.Context, args []string) int {
	const cmd = "reports.list"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only("limit", "offset"); err != nil {
		return e.badUsage(cmd, err.Error())
	}
	limit, offset, ok := e.pageArgs(cmd, parsed)
	if !ok {
		return 1
	}
	page, err := e.deps.Reports.ListOpen(ctx, user, limit, offset)
	if err != nil {
		return e.failErr(cmd, err)
	}
	out := make([]map[string]any, 0, len(page.Reports))
	for _, report := range page.Reports {
		out = append(out, reportView(report))
	}
	return e.ok(cmd, map[string]any{"total": page.Total, "reports": out})
}

func (e *Engine) reportsDismiss(ctx context.Context, args []string) int {
	const cmd = "reports.dismiss"
	user, ok := e.requireActor(ctx, cmd)
	if !ok || !e.requireMin(cmd, user, "moderator") {
		return 1
	}
	parsed, err := parseArgs(args)
	if err != nil {
		return e.badUsage(cmd, err.Error())
	}
	if err := parsed.Only(); err != nil || len(parsed.Pos) != 1 {
		return e.badUsage(cmd, "usage: kura agent reports dismiss <report-id>")
	}
	report, err := e.deps.Reports.Dismiss(ctx, user, parsed.Pos[0])
	if err != nil {
		return e.failErr(cmd, err)
	}
	return e.ok(cmd, reportView(report))
}

func reportView(report model.PostReport) map[string]any {
	return map[string]any{
		"id": report.ID, "post_id": report.PostID, "reporter_username": report.ReporterUsername,
		"reason": report.Reason, "note": report.Note, "status": report.Status,
	}
}
