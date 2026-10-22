-- name: GetSite :one
SELECT
    (SELECT value FROM site_settings WHERE key = 'site_name')::text AS name,
    (SELECT value FROM site_settings WHERE key = 'site_description')::text AS description;
