package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	udpBufferSize    = 2048
	inactivityPeriod = 60 * time.Second
)

type UDPClient struct {
	addr     *net.UDPAddr
	username string
	lastSeen time.Time
	lastMsg  time.Time
}

type UDPServer struct {
	conn       *net.UDPConn
	clients    map[string]*UDPClient
	mu         sync.Mutex
	broadcast  chan string
	register   chan *net.UDPAddr
	unregister chan *net.UDPAddr
}

func NewUDPServer(port string) (*UDPServer, error) {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	return &UDPServer{
		conn:       conn,
		clients:    make(map[string]*UDPClient),
		broadcast:  make(chan string, 100),
		register:   make(chan *net.UDPAddr),
		unregister: make(chan *net.UDPAddr),
	}, nil
}

func (s *UDPServer) logDelay(client *UDPClient, message string) {
	now := time.Now()
	if !client.lastMsg.IsZero() {
		delay := now.Sub(client.lastMsg).Milliseconds()
		log.Printf("[%s] Delay: %dms | %s", client.username, delay, message)
	}
	client.lastMsg = now
}

func (s *UDPServer) Run() {
	go s.handleBroadcasts()
	go s.handleRegistrations()
	go s.handleUnregistrations()
	go s.cleanupInactiveClients()

	buf := make([]byte, udpBufferSize)
	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}

		msg := strings.TrimSpace(string(buf[:n]))
		if msg == "" {
			continue
		}

		s.mu.Lock()
		client, exists := s.clients[addr.String()]
		if !exists {
			s.clients[addr.String()] = &UDPClient{
				addr:     addr,
				username: msg,
				lastSeen: time.Now(),
			}
			log.Printf("New client: %s (%s)", msg, addr)
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()

		client.lastSeen = time.Now()
		s.logDelay(client, msg)

		switch msg {
		case "/quit", "bye":
			s.unregister <- addr
		default:
			s.broadcast <- fmt.Sprintf("[%s] %s", client.username, msg)
		}
	}
}

func (s *UDPServer) handleBroadcasts() {
	for msg := range s.broadcast {
		s.mu.Lock()
		for _, client := range s.clients {
			_, err := s.conn.WriteToUDP([]byte(msg), client.addr)
			if err != nil {
				log.Printf("Send error to %s: %v", client.username, err)
			}
		}
		s.mu.Unlock()
	}
}

func (s *UDPServer) handleRegistrations() {
	for addr := range s.register {
		s.mu.Lock()
		if _, exists := s.clients[addr.String()]; !exists {
			s.clients[addr.String()] = &UDPClient{
				addr:     addr,
				lastSeen: time.Now(),
			}
		}
		s.mu.Unlock()
	}
}

func (s *UDPServer) handleUnregistrations() {
	for addr := range s.unregister {
		s.mu.Lock()
		if client, exists := s.clients[addr.String()]; exists {
			delete(s.clients, addr.String())
			log.Printf("Client left: %s", client.username)
		}
		s.mu.Unlock()
	}
}

func (s *UDPServer) cleanupInactiveClients() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for addr, client := range s.clients {
			if time.Since(client.lastSeen) > inactivityPeriod {
				delete(s.clients, addr)
				log.Printf("Timeout: %s", client.username)
			}
		}
		s.mu.Unlock()
	}
}

func main() {
	server, err := NewUDPServer("3001")
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
	defer server.conn.Close()
	log.Printf("UDP server running on :3001")
	server.Run()
}
