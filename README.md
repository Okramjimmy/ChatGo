# ChatGo 🚀

ChatGo is a production-grade, on-premise real-time messaging platform built with Golang. It provides a highly scalable and secure backend for enterprise chat applications, featuring WebSockets for real-time delivery, JWT for stateless authentication, and clean architecture principles.

## 🌟 Features

- **Real-Time Messaging**: Built-in WebSocket hub for instant message delivery, typing indicators, read receipts, and reactions.
- **Conversations**: Support for Direct Messages (1-on-1), Private Groups, and Public Channels.
- **Robust Authentication**: JWT-based authentication (short-lived access, long-lived refresh tokens) with multi-factor authentication (MFA/TOTP) support.
- **Role-Based Access Control**: Granular permissions via seeded roles (`member`, `admin`, `moderator`).
- **File Sharing**: Secure multipart file uploads with MIME type validation, size restrictions, and optional antivirus scanning hooks.
- **Full-Text Search**: Powered by PostgreSQL `pg_trgm` and `tsvector` for fast message and user discovery.
- **Presence Tracking**: Real-time online/offline status monitoring backed by Redis.
- **Audit Logging**: Comprehensive activity logs for security and compliance.
- **Production Ready**: Built-in rate limiting, Prometheus metrics, structured JSON logging (Zap), and CORS middleware.

---

## 🛠️ Technology Stack

- **Language**: Go (Golang)
- **Database**: PostgreSQL (pgxpool)
- **Cache & Pub/Sub**: Redis
- **Router**: `go-chi/chi`
- **Configuration**: Viper & Godotenv
- **WebSockets**: `gorilla/websocket`
- **Migrations**: `golang-migrate`

---

## ⚙️ Prerequisites

Before you begin, ensure you have the following installed on your system:
- **Go**: `1.20` or newer
- **PostgreSQL**: `14` or newer
- **Redis**: `6` or newer
- **Make**: Standard GNU Make

---

## 🚀 Installation & Setup

### 1. Clone the repository
```bash
git clone https://github.com/okrammeitei/chatgo.git
cd chatgo
```

### 2. Set up the Database & Redis
Ensure PostgreSQL and Redis are running. Create the database and user:

```bash
# Example using psql
psql postgres -c "CREATE USER chatgo WITH PASSWORD 'changeme';"
psql postgres -c "CREATE DATABASE chatgo OWNER chatgo;"
```

### 3. Environment Configuration
Copy the environment template and adjust the values if necessary (especially the JWT secret):

```bash
cp .env.example .env
```
> Note: By default, the `.env` file assumes PostgreSQL is on `localhost:5432` and Redis is on `localhost:6379`.

### 4. Build the Project
Use the provided `Makefile` to compile the server and migration binaries:

```bash
make build
```

### 5. Run Database Migrations
Apply the SQL schema migrations to your PostgreSQL database:

```bash
# Load env vars and run the migration tool
export $(grep -v '^#' .env | xargs) && make migrate
```

### 6. Start the Server

There are two ways to start the server:

**Option A: Development Mode (using `go run`)**
Start the development server using the Makefile:
```bash
export $(grep -v '^#' .env | xargs) && make run
```

**Option B: Production Mode (using compiled binaries)**
If you built the project in Step 4, you can run the compiled binaries directly from the `bin/` directory:
```bash
# Run the backend server
export $(grep -v '^#' .env | xargs) && ./bin/chatgo-server

# (Optional) Run migrations via compiled binary
export $(grep -v '^#' .env | xargs) && ./bin/chatgo-migrate up
```

The server will start on `http://localhost:8081` (or whichever port you specified in `.env`).

---

## 📖 How to Use

Once the server is running, you can interact with the REST API. For a complete list of endpoints and cURL examples, see the [example.md](example.md) file included in this repository.

### Quick Start Flow

1. **Register a User**: Create your first account using `POST /api/v1/users`.
2. **Login**: Get your JWT tokens using `POST /api/v1/auth/login`.
3. **Use the API**: Attach the token as a Bearer token in the `Authorization` header for all subsequent requests.
   ```http
   Authorization: Bearer <your_access_token>
   ```
4. **Connect to WebSocket**: Connect your client to `ws://localhost:8081/api/v1/ws` to start receiving real-time events (`message.new`, `typing.start`, `presence.update`, etc.).

---

## 🧪 Testing and Quality

Run the test suite and static analysis tools using Make:

```bash
# Run all tests
make test

# Run static analysis
make vet

# Format and tidy dependencies
make tidy
```

---

## 📦 Deployment

ChatGo is designed for on-premise deployment. A systemd unit file is provided in `deploy/systemd/chatgo.service` for easy deployment on Linux environments. 

1. Build and install the binaries: `make install`
2. Copy the unit file to `/etc/systemd/system/`
3. Configure your `/etc/chatgo/chatgo.env`
4. Start the service: `systemctl enable --now chatgo`
