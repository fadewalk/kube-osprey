# RBAC Management

RBAC management APIs are under `/api/v1/admin/roles/`.

These endpoints require an authenticated admin user, or an API key with the `admin` role.

## List roles

```http
GET /api/v1/admin/roles/
```

Example:

```bash
curl \
  -H "Authorization: kite1-adminsecret" \
  https://kube-osprey.example.com/api/v1/admin/roles/
```

Response example:

```json
{
  "roles": [
    {
      "id": 2,
      "name": "viewer",
      "description": "Viewer role with read-only access",
      "isSystem": true,
      "clusters": ["*"],
      "namespaces": ["*"],
      "resources": ["*"],
      "verbs": ["get", "log"],
      "assignments": []
    }
  ]
}
```

## Get a single role

```http
GET /api/v1/admin/roles/:id
```

## Create a role

```http
POST /api/v1/admin/roles/
Content-Type: application/json
```

Request body:

```json
{
  "name": "namespace-editor",
  "description": "Edit workloads in dev",
  "clusters": ["demo-cluster"],
  "namespaces": ["dev"],
  "resources": ["deployments", "pods", "services"],
  "verbs": ["get", "create", "update", "delete", "log"]
}
```

Example:

```bash
curl \
  -X POST \
  -H "Authorization: kite1-adminsecret" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"namespace-editor",
    "description":"Edit workloads in dev",
    "clusters":["demo-cluster"],
    "namespaces":["dev"],
    "resources":["deployments","pods","services"],
    "verbs":["get","create","update","delete","log"]
  }' \
  https://kube-osprey.example.com/api/v1/admin/roles/
```

## Update a role

```http
PUT /api/v1/admin/roles/:id
Content-Type: application/json
```

Request body uses the same fields as create.

## Delete a role

```http
DELETE /api/v1/admin/roles/:id
```

Example:

```bash
curl \
  -X DELETE \
  -H "Authorization: kite1-adminsecret" \
  https://kube-osprey.example.com/api/v1/admin/roles/3
```

Response example:

```json
{
  "success": true
}
```

## Assign a role

```http
POST /api/v1/admin/roles/:id/assign
Content-Type: application/json
```

Request body:

```json
{
  "subjectType": "user",
  "subject": "alice",
  "expiresAt": "2026-12-31T23:59:59Z"
}
```

Supported `subjectType` values in the current server implementation:

- `user`
- `group`

`expiresAt` is optional and takes an RFC 3339 timestamp. An expired assignment stops granting the role immediately on the next permission check. Assigning the same subject again updates the assignment, so you can extend or clear the expiry this way. See [Role Assignment Expiry](/config/rbac-config#role-assignment-expiry) for the UI flow.

Example:

```bash
curl \
  -X POST \
  -H "Authorization: kite1-adminsecret" \
  -H "Content-Type: application/json" \
  -d '{"subjectType":"group","subject":"platform-team"}' \
  https://kube-osprey.example.com/api/v1/admin/roles/3/assign
```

## Unassign a role

```http
DELETE /api/v1/admin/roles/:id/assign?subjectType=user&subject=alice
```

Example:

```bash
curl \
  -X DELETE \
  -H "Authorization: kite1-adminsecret" \
  "https://kube-osprey.example.com/api/v1/admin/roles/3/assign?subjectType=group&subject=platform-team"
```
