package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxMessageSize   = 1024
	inactivityPeriod = 60 * time.Second
	logDir           = "client_logs"
)

var (
	clientCount     int
	clientCountLock sync.Mutex
)

type Client struct {
	conn        net.Conn
	name        string
	logFile     *os.File
	lastSeen    time.Time
	lastMsgTime time.Time
}

type Server struct {
	clients    map[net.Conn]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan string
	mu         sync.Mutex
}

func NewServer() *Server {
	return &Server{
		clients:    make(map[net.Conn]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan string),
	}
}

func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client.conn] = client
			s.mu.Unlock()
			logEvent(fmt.Sprintf("Client connected: %s", client.name))
		case client := <-s.unregister:
			s.mu.Lock()
			if c, ok := s.clients[client.conn]; ok {
				c.logFile.Close()
				delete(s.clients, client.conn)
				client.conn.Close()
				logEvent(fmt.Sprintf("Client disconnected: %s", client.name))
			}
			s.mu.Unlock()
		case message := <-s.broadcast:
			s.mu.Lock()
			for _, client := range s.clients {
				_, err := fmt.Fprintln(client.conn, message)
				if err != nil {
					log.Printf("Error sending message to %s: %v", client.name, err)
				}
			}
			s.mu.Unlock()
		}
	}
}

func handleConnection(conn net.Conn, server *Server) {
	defer func() {
		server.unregister <- &Client{conn: conn}
	}()

	clientAddr := conn.RemoteAddr().String()
	os.MkdirAll(logDir, 0755)
	safeFileName := strings.ReplaceAll(clientAddr, ":", "_") + ".log"
	logFilePath := filepath.Join(logDir, safeFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(conn, "Server error: unable to open log file\n")
		conn.Close()
		return
	}

	client := &Client{
		conn:        conn,
		name:        clientAddr,
		logFile:     logFile,
		lastSeen:    time.Now(),
		lastMsgTime: time.Now(),
	}
	server.register <- client

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, maxMessageSize), maxMessageSize)

	timer := time.NewTimer(inactivityPeriod)
	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(inactivityPeriod)
	}

	done := make(chan bool)

	go func() {
		for scanner.Scan() {
			resetTimer()
			input := strings.TrimSpace(scanner.Text())

			now := time.Now()
			delay := now.Sub(client.lastMsgTime).Milliseconds()
			client.lastMsgTime = now

			logEvent(fmt.Sprintf("Delay since last message from %s: %d ms", client.name, delay))

			if len(input) > maxMessageSize {
				conn.Write([]byte("Whoa there, message too long.\n"))
				input = input[:maxMessageSize]
			}

			client.logFile.WriteString(fmt.Sprintf(
				"[%s] Delay: %dms | %s\n",
				now.Format(time.RFC3339),
				delay,
				input,
			))

			switch {
			case input == "":
				conn.Write([]byte("Silence is golden...\n"))
			case input == "bye" || input == "/quit":
				conn.Write([]byte("Goodbye traveler. May your packets be swift!\n"))
				done <- true
				return
			case input == "/time":
				conn.Write([]byte("Server Time: " + time.Now().Format("15:04:05 MST") + "\n"))
			case input == "/date":
				conn.Write([]byte("Today's Date: " + time.Now().Format("Monday, Jan 2, 2006") + "\n"))
			case input == "/nocknock":
				conn.Write([]byte("Who's there? Not your GPA, go study!\n"))
			case input == "/clients":
				clientCountLock.Lock()
				count := clientCount
				clientCountLock.Unlock()
				conn.Write([]byte(fmt.Sprintf("There are currently %d humans connected.\n", count)))
			case input == "/help":
				conn.Write([]byte("Available commands:\n" +
					"/name [your_name] - Sets your display name\n" +
					"/echo [message] - Echoes back your message\n" +
					"/time - Shows current server time\n" +
					"/date - Shows current server date\n" +
					"/nocknock - Joke command\n" +
					"/clients - Number of connected clients\n" +
					"/quit or bye - Disconnects you\n"))
			case strings.HasPrefix(input, "/name "):
				newName := strings.TrimSpace(strings.TrimPrefix(input, "/name "))
				if newName != "" {
					oldName := client.name
					client.name = newName
					conn.Write([]byte("Username set to " + newName + "\n"))
					server.broadcast <- fmt.Sprintf("* %s is now known as %s *", oldName, newName)
				}
			case strings.HasPrefix(input, "/echo "):
				conn.Write([]byte("Echo: " + strings.TrimPrefix(input, "/echo ") + "\n"))
			default:
				server.broadcast <- fmt.Sprintf("%s: %s", client.name, input)
			}
		}
		if err := scanner.Err(); err != nil {
			logEvent(fmt.Sprintf("Error reading from client %s: %v", client.name, err))
		}
		done <- true
	}()

	clientCountLock.Lock()
	clientCount++
	clientCountLock.Unlock()

	select {
	case <-timer.C:
		conn.Write([]byte("Disconnected due to inactivity\n"))
		logEvent(fmt.Sprintf("Client disconnected (timeout): %s", clientAddr))
	case <-done:
		logEvent(fmt.Sprintf("Client disconnected: %s", clientAddr))
	}

	clientCountLock.Lock()
	clientCount--
	clientCountLock.Unlock()
}

func logEvent(message string) {
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Printf("[%s] %s\n", timestamp, message)
}

func main() {
	port := "3000"
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Error starting TCP server: %v", err)
	}
	defer listener.Close()
	logEvent(fmt.Sprintf("TCP chat server started on :%s", port))

	server := NewServer()
	go server.Run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go handleConnection(conn, server)
	}
}
