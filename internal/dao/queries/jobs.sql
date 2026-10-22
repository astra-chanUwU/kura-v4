-- name: EnqueueJob :one
INSERT INTO jobs (kind, payload) VALUES ($1, $2) RETURNING id;

-- name: ClaimJob :one
WITH next_job AS (
    SELECT id FROM jobs
    WHERE kind = ANY(sqlc.arg(kinds)::text[])
      AND attempts < max_attempts
      AND ((state = 'queued' AND available_at <= now())
        OR (state = 'running' AND lease_until < now()))
    ORDER BY available_at, id
    FOR UPDATE SKIP LOCKED LIMIT 1
)
UPDATE jobs SET state = 'running', attempts = attempts + 1,
    lease_until = now() + interval '60 seconds', lease_token = sqlc.arg(token), updated_at = now()
FROM next_job WHERE jobs.id = next_job.id
RETURNING jobs.id, jobs.kind, jobs.payload, jobs.attempts, jobs.max_attempts;

-- name: CompleteJob :execrows
UPDATE jobs SET state = 'complete', lease_until = NULL, lease_token = NULL, updated_at = now()
WHERE id = $1 AND lease_token = $2 AND state = 'running';

-- name: FailJob :execrows
UPDATE jobs SET
    state = CASE WHEN attempts >= max_attempts THEN 'failed' ELSE 'queued' END,
    available_at = now() + interval '5 seconds' * attempts,
    last_error = sqlc.arg(error_message), lease_until = NULL, lease_token = NULL, updated_at = now()
WHERE id = sqlc.arg(id) AND lease_token = sqlc.arg(token) AND state = 'running';

-- name: ReapExhaustedJobs :exec
UPDATE jobs SET state = 'failed', last_error = 'worker lease expired after final attempt',
    lease_until = NULL, lease_token = NULL, updated_at = now()
WHERE state = 'running' AND lease_until < now() AND attempts >= max_attempts;
