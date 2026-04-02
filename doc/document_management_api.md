# Document Management API

The document management system allows uploading, downloading, listing, updating, and deleting documents. Documents can be organized into **categories** that are fully user-manageable per club.

## Authentication

All endpoints require a valid JWT token in the `Authorization: Bearer <token>` header.
The user must have the appropriate `documents:read`, `documents:write`, or `documents:delete` permission via their assigned roles.

---

## Document Categories

Categories are per-club and allow flexible organization of documents. Five default categories are seeded on migration:
- **protocols** — Meeting protocols and minutes
- **contracts** — Contracts and agreements
- **invoices** — Invoices and billing documents
- **correspondence** — Letters and correspondence
- **miscellaneous** — Other documents

Clubs can create, rename, reorder, and delete categories at any time.

### List Categories

- **URL:** `GET /api/v1/document-categories`
- **Permission:** `documents:read`
- **Response:** Array of DocumentCategory objects, ordered by `sort_order` then `name`.

**Response Example:**
```json
[
  {
    "id": "a1b2c3d4-...",
    "club_id": "...",
    "name": "protocols",
    "description": "Meeting protocols and minutes",
    "sort_order": 1,
    "created_at": "2026-03-24T10:00:00Z",
    "updated_at": "2026-03-24T10:00:00Z"
  },
  {
    "id": "e5f6g7h8-...",
    "club_id": "...",
    "name": "contracts",
    "description": "Contracts and agreements",
    "sort_order": 2,
    "created_at": "2026-03-24T10:00:00Z",
    "updated_at": "2026-03-24T10:00:00Z"
  }
]
```

### Create Category

- **URL:** `POST /api/v1/document-categories`
- **Permission:** `documents:write`
- **Content-Type:** `application/json`

**Request Body:**
```json
{
  "name": "board-decisions",
  "description": "Decisions of the board of directors",
  "sort_order": 6
}
```

| Field         | Type   | Required | Description                          |
|---------------|--------|----------|--------------------------------------|
| `name`        | string | yes      | Unique name within the club          |
| `description` | string | no       | Human-readable description           |
| `sort_order`  | int    | no       | Display order (default: 0)           |

**Response:** `201 Created` — The created DocumentCategory object.

### Update Category

- **URL:** `PUT /api/v1/document-categories/{id}`
- **Permission:** `documents:write`
- **Content-Type:** `application/json`

**Request Body:**
```json
{
  "name": "board-decisions",
  "description": "Updated description",
  "sort_order": 3
}
```

**Response:** `200 OK` — The updated DocumentCategory object.

### Delete Category

- **URL:** `DELETE /api/v1/document-categories/{id}`
- **Permission:** `documents:delete`
- **Response:** `204 No Content`

> **Note:** When a category is deleted, documents that were assigned to it will have their `category_id` set to `NULL` (uncategorized) due to the `ON DELETE SET NULL` constraint.

---

## Documents

### Upload Document

- **URL:** `POST /api/v1/documents`
- **Permission:** `documents:write`
- **Content-Type:** `multipart/form-data`
- **Max file size:** 10 MB

**Form Fields:**

| Field         | Type   | Required | Description                                      |
|---------------|--------|----------|--------------------------------------------------|
| `file`        | file   | yes      | The file to upload                                |
| `category_id` | string | no       | UUID of the category to assign the document to    |
| `description` | string | no       | A short description of the document               |

**Example (curl):**
```bash
curl -X POST http://localhost:8580/api/v1/documents \
  -H "Authorization: Bearer <token>" \
  -F "file=@protocol_2026-03-15.pdf" \
  -F "category_id=a1b2c3d4-..." \
  -F "description=Board meeting protocol March 2026"
```

**Response:** `201 Created`
```json
{
  "id": "d1e2f3a4-...",
  "club_id": "...",
  "name": "protocol_2026-03-15.pdf",
  "category_id": "a1b2c3d4-...",
  "description": "Board meeting protocol March 2026",
  "created_at": "2026-03-24T12:00:00Z",
  "updated_at": "2026-03-24T12:00:00Z"
}
```

### List Documents

- **URL:** `GET /api/v1/documents`
- **Permission:** `documents:read`
- **Query Parameters:**

| Parameter     | Type   | Required | Description                                  |
|---------------|--------|----------|----------------------------------------------|
| `category_id` | string | no       | Filter documents by category UUID            |

**Response:** Array of document objects (without file content), including `category_name`:
```json
[
  {
    "id": "d1e2f3a4-...",
    "club_id": "...",
    "name": "protocol_2026-03-15.pdf",
    "category_id": "a1b2c3d4-...",
    "description": "Board meeting protocol March 2026",
    "created_at": "2026-03-24T12:00:00Z",
    "updated_at": "2026-03-24T12:00:00Z",
    "category_name": "protocols"
  }
]
```

### Update Document (Metadata)

- **URL:** `PUT /api/v1/documents/{id}`
- **Permission:** `documents:write`
- **Content-Type:** `application/json`

Updates the document name, category, and description. Does **not** replace the file content.

**Request Body:**
```json
{
  "name": "protocol_2026-03-15_v2.pdf",
  "category_id": "a1b2c3d4-...",
  "description": "Updated protocol"
}
```

| Field         | Type    | Required | Description                                      |
|---------------|---------|----------|--------------------------------------------------|
| `name`        | string  | yes      | New display name for the document                |
| `category_id` | string  | no       | UUID of category, or `null`/empty to uncategorize |
| `description` | string  | no       | Updated description                              |

**Response:** `200 OK` — The updated document metadata.

### Download Document

- **URL:** `GET /api/v1/documents/{id}/download`
- **Permission:** `documents:read`
- **Response:** Binary file download with `Content-Disposition: attachment` header.

### Delete Document

- **URL:** `DELETE /api/v1/documents/{id}`
- **Permission:** `documents:delete`
- **Response:** `204 No Content`

---

## Data Model

### DocumentCategory

| Field         | Type        | Description                      |
|---------------|-------------|----------------------------------|
| `id`          | UUID        | Primary key                      |
| `club_id`     | UUID        | Owning club                      |
| `name`        | string      | Category name (unique per club)  |
| `description` | string/null | Human-readable description       |
| `sort_order`  | int         | Display ordering                 |
| `created_at`  | timestamp   | Creation timestamp               |
| `updated_at`  | timestamp   | Last modification timestamp      |

### Document

| Field         | Type        | Description                                  |
|---------------|-------------|----------------------------------------------|
| `id`          | UUID        | Primary key                                  |
| `club_id`     | UUID        | Owning club                                  |
| `name`        | string      | File name                                    |
| `content`     | bytes       | File content (not returned in list queries)  |
| `category_id` | UUID/null   | Optional category reference                  |
| `description` | string/null | Optional description                         |
| `created_at`  | timestamp   | Creation timestamp                           |
| `updated_at`  | timestamp   | Last modification timestamp                  |

---

## Permissions & Roles

| Permission         | Roles                                                   |
|--------------------|---------------------------------------------------------|
| `documents:read`   | `documents_viewer`, `documents_writer`, `documents_manager`, `admin` |
| `documents:write`  | `documents_writer`, `documents_manager`, `admin`        |
| `documents:delete` | `documents_manager`, `admin`                            |
