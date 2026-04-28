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

---

## 2. Authentication & Sessions

### Refresh Token
Get a new access token using a refresh token.
```bash
curl -X POST http://localhost:8081/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "<your_refresh_token>"
  }'
```

### Logout
Revoke the current session.
```bash
curl -X POST http://localhost:8081/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

### Get Active Sessions
```bash
curl -X GET http://localhost:8081/api/v1/auth/sessions \
  -H "Authorization: Bearer $TOKEN"
```

---

## 3. Users

### Get Current User Profile
```bash
curl -X GET http://localhost:8081/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

### List Users (Search)
```bash
curl -X GET "http://localhost:8081/api/v1/users?search=john" \
  -H "Authorization: Bearer $TOKEN"
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

### List My Conversations
```bash
curl -X GET http://localhost:8081/api/v1/conversations \
  -H "Authorization: Bearer $TOKEN"
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

### List Messages in a Conversation
```bash
curl -X GET "http://localhost:8081/api/v1/conversations/<conv_uuid>/messages?limit=50" \
  -H "Authorization: Bearer $TOKEN"
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

### Mark a Message as Read
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages/<msg_uuid>/read \
  -H "Authorization: Bearer $TOKEN"
```

### Send Typing Indicator
```bash
curl -X POST http://localhost:8081/api/v1/conversations/<conv_uuid>/messages/typing \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_typing": true
  }'
```

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

### Download a File
```bash
curl -X GET http://localhost:8081/api/v1/files/<file_uuid>/download \
  -H "Authorization: Bearer $TOKEN" --output downloaded_image.png
```

---

## 7. Notifications

### Get Unread Notification Count
```bash
curl -X GET http://localhost:8081/api/v1/notifications/unread-count \
  -H "Authorization: Bearer $TOKEN"
```

### List Notifications
```bash
curl -X GET http://localhost:8081/api/v1/notifications \
  -H "Authorization: Bearer $TOKEN"
```

### Mark All as Read
```bash
curl -X POST http://localhost:8081/api/v1/notifications/read-all \
  -H "Authorization: Bearer $TOKEN"
```

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

### Get Bulk Presence
```bash
curl -X POST http://localhost:8081/api/v1/presence/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": ["<uuid_1>", "<uuid_2>"]
  }'
```

---

## 9. Search

### Global Search
Resource types can be `message`, `user`, or `channel`. Omit type to search all.
```bash
curl -X GET "http://localhost:8081/api/v1/search?q=project&type=message" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 10. WebSocket

Connect to real-time events. The server requires an `Authorization: Bearer` header, but since browser Native WebSockets do not support headers, you typically pass the token in standard WebSocket connection procedures (if implemented) or handle it strictly via secure cookies/proxy in the frontend.

**Endpoint URL:**
`ws://localhost:8081/api/v1/ws`

*The WebSocket automatically updates the user presence to `online` on connection, and `offline` on disconnect.*
