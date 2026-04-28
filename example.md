# ChatGo API Documentation & Examples

Base URL: `http://localhost:8081`

All protected routes require an `Authorization` header with a valid JWT Bearer token:
`Authorization: Bearer <your_access_token>`

---

## 1. Public Routes

### Health Check
Check if the server is running.
```bash
curl -X GET http://localhost:8081/health
```
**Response (200 OK):**
```json
{
  "status": "ok"
}
```

### Register User
Create a new user account.
```bash
curl -X POST http://localhost:8081/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securepassword123",
    "display_name": "John Doe"
  }'
```
**Response (201 Created):**
```json
{
  "id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
  "username": "johndoe",
  "email": "john@example.com",
  "display_name": "John Doe",
  "avatar_url": "",
  "status": "active",
  "mfa_enabled": false,
  "created_at": "2023-10-25T10:00:00Z"
}
```

### Login
Authenticate and get access/refresh tokens.
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "password": "securepassword123"
  }'
```
> **Note:** Copy the `access_token` from the response to use in the `Authorization` header for the protected routes below.

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1...",
  "refresh_token": "dGVzdF9yZWZyZXNo...",
  "user": {
    "id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
    "username": "johndoe",
    "email": "john@example.com",
    "display_name": "John Doe"
  }
}
```

---

## 2. Authentication & Sessions

### Refresh Token
Get a new access token using a refresh token.
```bash
curl -X POST http://localhost:8081/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "dGVzdF9yZWZyZXNo..."
  }'
```
**Response (200 OK):**
*(Same structure as Login response)*

### Logout
Revoke the current session.
```bash
curl -X POST http://localhost:8081/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```
**Response (204 No Content)**

### Get Active Sessions
```bash
curl -X GET http://localhost:8081/api/v1/auth/sessions \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
[
  {
    "id": "11111111-2222-3333-4444-555555555555",
    "user_id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0...",
    "created_at": "2023-10-25T10:00:00Z",
    "expires_at": "2023-11-01T10:00:00Z"
  }
]
```

---

## 3. Users

### Get Current User Profile
```bash
curl -X GET http://localhost:8081/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
*(Returns the User object similar to Registration)*

### List Users (Search)
```bash
curl -X GET "http://localhost:8081/api/v1/users?search=john" \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
      "username": "johndoe",
      "display_name": "John Doe",
      "status": "active"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

---

## 4. Conversations

### Create a Direct Message or Group
Types can be `direct`, `group`, or `channel`.
```bash
curl -X POST http://localhost:8081/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "group",
    "name": "Engineering Team",
    "participant_ids": ["<user_uuid_1>", "<user_uuid_2>"]
  }'
```
**Response (201 Created):**
```json
{
  "id": "c1c1c1c1-c1c1-c1c1-c1c1-c1c1c1c1c1c1",
  "type": "group",
  "name": "Engineering Team",
  "creator_id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
  "created_at": "2023-10-25T10:05:00Z"
}
```

### List My Conversations
```bash
curl -X GET http://localhost:8081/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
[
  {
    "id": "c1c1c1c1-c1c1-c1c1-c1c1-c1c1c1c1c1c1",
    "type": "group",
    "name": "Engineering Team"
  }
]
```

### Add a Member to a Conversation
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "<new_user_uuid>",
    "role": "member"
  }'
```
**Response (204 No Content)**

---

## 5. Messages

*Messages are nested under their respective conversation.*

### Send a Message
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello team, let us discuss the new features!",
    "content_type": "text"
  }'
```
**Response (201 Created):**
```json
{
  "id": "m1m1m1m1-m1m1-m1m1-m1m1-m1m1m1m1m1m1",
  "conversation_id": "c1c1c1c1-c1c1-c1c1-c1c1-c1c1c1c1c1c1",
  "sender_id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
  "content": "Hello team, let us discuss the new features!",
  "content_type": "text",
  "created_at": "2023-10-25T10:10:00Z"
}
```

### List Messages in a Conversation
```bash
curl -X GET "http://localhost:8081/api/v1/conversations/<conv_uuid>/messages?limit=50" \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
[
  {
    "id": "m1m1m1m1-m1m1-m1m1-m1m1-m1m1m1m1m1m1",
    "content": "Hello team, let us discuss the new features!",
    "sender_id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a"
  }
]
```

### Add a Reaction to a Message
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages/<msg_uuid>/reactions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emoji": "👍"
  }'
```
**Response (204 No Content)**

### Mark a Message as Read
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages/<msg_uuid>/read \
  -H "Authorization: Bearer $TOKEN"
```
**Response (204 No Content)**

### Send Typing Indicator
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages/typing \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_typing": true
  }'
```
**Response (204 No Content)**

---

## 6. Files

### Upload a File
This uses `multipart/form-data`.
```bash
curl -X POST http://localhost:8081/api/v1/files \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/your/image.png" \
  -F "conversation_id=<optional_conv_uuid>"
```
**Response (201 Created):**
```json
{
  "file": {
    "id": "f1f1f1f1-f1f1-f1f1-f1f1-f1f1f1f1f1f1",
    "name": "f1f1f1f1-f1f1-f1f1-f1f1-f1f1f1f1f1f1.png",
    "original_name": "image.png",
    "mime_type": "image/png",
    "size": 1048576,
    "uploader_id": "e0b5f1f9-8b8a-4b9a-8b8a-e0b5f1f98b8a",
    "created_at": "2023-10-25T10:15:00Z"
  },
  "download_url": "/api/v1/files/f1f1f1f1-f1f1-f1f1-f1f1-f1f1f1f1f1f1/download"
}
```

### Download a File
```bash
curl -X GET http://localhost:8081/api/v1/files/<file_uuid>/download \
  -H "Authorization: Bearer $TOKEN" --output downloaded_image.png
```
**Response (200 OK):**
*(Binary file content stream)*

---

## 7. Notifications

### Get Unread Notification Count
```bash
curl -X GET http://localhost:8081/api/v1/notifications/unread-count \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
{
  "count": 5
}
```

### List Notifications
```bash
curl -X GET http://localhost:8081/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "n1n1n1n1-n1n1-n1n1-n1n1-n1n1n1n1n1n1",
      "type": "message",
      "title": "New message in Engineering Team",
      "body": "Hello team, let us discuss...",
      "is_read": false,
      "created_at": "2023-10-25T10:10:00Z"
    }
  ],
  "total": 5
}
```

### Mark All as Read
```bash
curl -X POST http://localhost:8081/api/v1/notifications/read-all \
  -H "Authorization: Bearer $TOKEN"
```
**Response (204 No Content)**

---

## 8. Presence

### Set Own Presence
Statuses: `online`, `offline`, `away`, `busy`.
```bash
curl -X PUT http://localhost:8081/api/v1/presence/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "away"
  }'
```
**Response (204 No Content)**

### Get Bulk Presence
```bash
curl -X POST http://localhost:8081/api/v1/presence/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": ["<uuid_1>", "<uuid_2>"]
  }'
```
**Response (200 OK):**
```json
[
  {
    "user_id": "<uuid_1>",
    "status": "online",
    "last_seen": "2023-10-25T10:12:00Z"
  },
  {
    "user_id": "<uuid_2>",
    "status": "away",
    "last_seen": "2023-10-25T09:00:00Z"
  }
]
```

---

## 9. Search

### Global Search
Resource types can be `message`, `user`, or `channel`. Omit type to search all.
```bash
curl -X GET "http://localhost:8081/api/v1/search?q=project&type=message" \
  -H "Authorization: Bearer $TOKEN"
```
**Response (200 OK):**
```json
{
  "results": [
    {
      "id": "m1m1m1m1-m1m1-m1m1-m1m1-m1m1m1m1m1m1",
      "resource_type": "message",
      "title": "We need to talk about the project",
      "excerpt": "We need to talk about the project...",
      "score": 0.95
    }
  ],
  "total": 1
}
```

---

## 10. WebSocket

Connect to real-time events. The server requires an `Authorization: Bearer` header, but since browser Native WebSockets do not support headers, you typically pass the token in standard WebSocket connection procedures (if implemented) or handle it strictly via secure cookies/proxy in the frontend.

**Endpoint URL:**
`ws://localhost:8081/api/v1/ws`

*The WebSocket automatically updates the user presence to `online` on connection, and `offline` on disconnect.*
