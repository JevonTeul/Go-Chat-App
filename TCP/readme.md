# TCP Chat Application with Delay Logging

A simple TCP-based chat server and client written in Go, with message delay tracking.

## Features
- Real-time chat between multiple clients
- Message delay logging (server & client)
- Inactivity timeout (60s)
- Command support (`/time`, `/date`, `/quit`, etc.)
- Client logs stored in `client_logs/`

## How to Run

### 1. Start the Server
```bash
cd server/
go run main.go
```
### 2. Start multiple clients
```bash
cd client/
go run main.go
