package model

import "errors"

// ErrNotFound is a missing row. Repositories return it instead of sql.ErrNoRows
// so services do not import database/sql. Services return this same error.
// HTTP and the agent CLI map it to 404 / not_found and the sentence "Not found."
var ErrNotFound = errors.New("not found")

// ErrEmpty is a query that matched nothing when that is a normal outcome,
// such as a job queue with nothing to claim. It is not a missing entity.
var ErrEmpty = errors.New("empty")
