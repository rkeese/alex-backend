# Frontend Security Integration Documentation

## Overview
This document describes how permissions and roles are handled between the Backend (Go) and Frontend (TypeScript/React).

## 1. The Token Structure
When a user logs in via `/api/v1/auth/login` or registers via `/api/v1/auth/register`, they receive a JWT token. This token now contains a custom claim `roles` which is an ARRAY of objects.

**JWT Payload Example:**
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alextester@example.com",
  "roles": [
    {
      "club_id": "00000000-0000-0000-0000-000000000000",
      "role": "members_writer"
    },
    {
      "club_id": "00000000-0000-0000-0000-000000000000",
      "role": "finance_viewer"
    }
  ],
  "exp": 1735689600
}
```

## 2. Frontend Handling Logic

The frontend authentication service (e.g., `auth.ts`) MUST:

1.  **Parse the JWT:** Decode the token to access the payload.
2.  **Extract Roles:** Look for the key `roles`.
3.  **Handle Objects:** Note that `roles` is an array of objects `{ club_id: string, role: string }`.
4.  **Flatten/Map:** For simple checks, you may want to map this to a list of strings if you only care about the role name.

**TypeScript Example (Recommended Implementation):**

```typescript
interface RoleObject {
  club_id: string;
  role: string;
}

// Inside your parsing logic
const decodedToken = jwtDecode(token);
// SAFELY extract roles, handling both potential formats (legacy string vs new object)
const userRoles = (decodedToken.roles || []).map((r: any) => {
    // If backend sends object { role: "name" }
    if (typeof r === 'object' && r !== null && r.role) {
        return r.role.toLowerCase(); 
    }
    // If backend sent string "name"
    return String(r).toLowerCase();
});

// Result: ['members_writer', 'finance_viewer']
```

## 3. Permission Deduction (Frontend)

Since the backend does not send granular permissions (like `members:read`) in the token to keep it small, the Frontend must **deduce** permissions from the Role Name.

| Role Name (in Token) | Implied Permissions (Frontend) |
| :--- | :--- |
| `members_viewer` | `members:read` |
| `members_writer` | `members:read`, `members:write` |
| `members_manager` | `members:read`, `members:write`, `members:delete` |
| `admin` | *ALL PERMISSIONS* |

**Logic Example:**

```typescript
function hasPermission(userRoles: string[], permission: string): boolean {
    if (userRoles.includes('admin')) return true;

    const [area, action] = permission.split(':'); // e.g., "members", "write"
    
    // Check for explicit role matches
    // e.g. members_writer has 'write' so it implies 'read' too
    
    // Simple check: "members_writer"
    const roleForArea = userRoles.find(r => r.startsWith(area));
    if (!roleForArea) return false;

    // Logic: members_manager > members_writer > members_viewer
    if (action === 'read') return true; // Any role (viewer, writer, manager) can read
    if (action === 'write') return roleForArea.includes('writer') || roleForArea.includes('manager');
    if (action === 'delete') return roleForArea.includes('manager');

    return false;
}
```

## 4. Debugging

If a user reports missing rights:
1.  Ask them to **Logout and Login**. This refreshes the token.
2.  Check the Backend Database:
    ```sql
    SELECT r.name 
    FROM user_roles ur 
    JOIN roles r ON ur.role_id = r.id 
    WHERE ur.user_id = 'USER_UUID';
    ```
3.  Inspect the user's Token (using a JWT debugger) to ensure the `roles` array is present and populated as expected.
