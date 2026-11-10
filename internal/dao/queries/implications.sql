-- name: ListTagImplications :many
SELECT antecedent, consequent
FROM tag_implications
ORDER BY antecedent, consequent;

-- name: InsertTagImplication :exec
INSERT INTO tag_implications (antecedent, consequent, created_by)
VALUES (sqlc.arg(antecedent), sqlc.arg(consequent), sqlc.arg(created_by));

-- name: DeleteTagImplication :execrows
DELETE FROM tag_implications
WHERE antecedent = sqlc.arg(antecedent) AND consequent = sqlc.arg(consequent);

-- name: BackfillImplication :exec
INSERT INTO post_tags (post_id, tag)
SELECT pt.post_id, sqlc.arg(consequent)
FROM post_tags pt
WHERE pt.tag = sqlc.arg(antecedent)
ON CONFLICT (post_id, tag) DO NOTHING;
