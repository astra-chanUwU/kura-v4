package model

import "errors"

// ErrNotFound is a missing row. Repositories return it instead of sql.ErrNoRows
// so services do not import database/sql. Callers map it to their own sentinel
// before it reaches HTTP.
var ErrNotFound = errors.New("not found")

// ErrEmpty is a query that matched nothing when that is a normal outcome,
// such as a job queue with nothing to claim. It is not a missing entity.
var ErrEmpty = errors.New("empty")
