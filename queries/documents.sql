-- =====================
-- Document Categories
-- =====================

-- name: CreateDocumentCategory :one
INSERT INTO document_categories (club_id, name, description, sort_order)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListDocumentCategories :many
SELECT * FROM document_categories
WHERE club_id = $1
ORDER BY sort_order, name;

-- name: GetDocumentCategory :one
SELECT * FROM document_categories
WHERE id = $1 AND club_id = $2;

-- name: UpdateDocumentCategory :one
UPDATE document_categories
SET name = $3, description = $4, sort_order = $5, updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING *;

-- name: DeleteDocumentCategory :exec
DELETE FROM document_categories
WHERE id = $1 AND club_id = $2;

-- =====================
-- Documents
-- =====================

-- name: CreateDocument :one
INSERT INTO documents (
  club_id, name, content, category_id, description
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING id, club_id, name, category_id, description, created_at, updated_at;

-- name: ListDocuments :many
SELECT d.id, d.club_id, d.name, d.category_id, d.description, d.created_at, d.updated_at,
       dc.name AS category_name
FROM documents d
LEFT JOIN document_categories dc ON d.category_id = dc.id
WHERE d.club_id = $1
ORDER BY d.created_at DESC;

-- name: ListDocumentsByCategory :many
SELECT d.id, d.club_id, d.name, d.category_id, d.description, d.created_at, d.updated_at,
       dc.name AS category_name
FROM documents d
LEFT JOIN document_categories dc ON d.category_id = dc.id
WHERE d.club_id = $1 AND d.category_id = $2
ORDER BY d.created_at DESC;

-- name: GetDocument :one
SELECT * FROM documents
WHERE id = $1 AND club_id = $2;

-- name: UpdateDocument :one
UPDATE documents
SET name = $3, category_id = $4, description = $5, updated_at = NOW()
WHERE id = $1 AND club_id = $2
RETURNING id, club_id, name, category_id, description, created_at, updated_at;

-- name: DeleteDocument :exec
DELETE FROM documents
WHERE id = $1 AND club_id = $2;
