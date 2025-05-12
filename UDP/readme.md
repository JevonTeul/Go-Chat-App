# UDP Chat Application with Delay Tracking
A lightweight UDP-based chat system featuring real-time message delay tracking, designed for learning Go networking fundamentals.

## Features ✨

- **UDP Protocol**: Low-latency communication without connection overhead
- **Delay Metrics**: Tracks time between messages (server & client-side)
- **User Identities**: Unique usernames for participant identification
- **Inactivity Timeout**: Automatic client cleanup after 60s idle
- **Concurrent Safe**: Mutex-protected client registry
- **Command Support**: (`/time`, `/date`, `/quit`, etc.)

## How to run
-**open server in a new Terminal**
```bash
cd server
go run server.go
```
- **open client in a new terminal**
```bash
cd client
go run client.go
```

