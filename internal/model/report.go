package model

import "time"

// PostReport is one member report on a post. Dismissing it records who
// closed it and when. It does not hide or delete the post.
type PostReport struct {
	ID               string
	PostID           string
	ReporterID       string
	ReporterUsername string
	Reason           string
	Note             string
	Status           string
	CreatedAt        time.Time
	ResolvedAt       time.Time
	ResolverID       string
}

// PostReportPage is one page of open reports, newest first.
type PostReportPage struct {
	Total   int64
	Reports []PostReport
}

// ReportPostAccess is the slice of a post used to decide whether a
// viewer may know that post exists. A miss is not found.
type ReportPostAccess struct {
	OwnerID string
	Rating  string
	Hidden  bool
	Deleted bool
}
