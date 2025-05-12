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
	lastSeen time.Time
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
	server := &UDPServer{
		conn:       conn,
		clients:    make(map[string]*UDPClient),
		broadcast:  make(chan string),
		register:   make(chan *net.UDPAddr),
		unregister: make(chan *net.UDPAddr),
	}
	return server, nil
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
			log.Printf("Error reading UDP packet: %v", err)
			continue
		}
		message := strings.TrimSpace(string(buf[:n]))
		if message == "" {
			continue
		}

		s.register <- addr

		if message == "/quit" || message == "bye" {
			s.unregister <- addr
			continue
		}

		s.mu.Lock()
		client, ok := s.clients[addr.String()]
		if ok {
			client.lastSeen = time.Now()
		}
		s.mu.Unlock()

		// Add timestamp to message before broadcasting
		timestamp := time.Now().Format("15:04:05")
		broadcastMsg := fmt.Sprintf("[%s] %s: %s", timestamp, addr.String(), message)
		s.broadcast <- broadcastMsg
	}
}

func (s *UDPServer) handleBroadcasts() {
	for msg := range s.broadcast {
		s.mu.Lock()
		for _, client := range s.clients {
			_, err := s.conn.WriteToUDP([]byte(msg), client.addr)
			if err != nil {
				log.Printf("Error sending to %s: %v", client.addr.String(), err)
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
			log.Printf("Client registered: %s", addr.String())
		}
		s.mu.Unlock()
	}
}

func (s *UDPServer) handleUnregistrations() {
	for addr := range s.unregister {
		s.mu.Lock()
		if _, exists := s.clients[addr.String()]; exists {
			delete(s.clients, addr.String())
			log.Printf("Client unregistered: %s", addr.String())
		}
		s.mu.Unlock()
	}
}

func (s *UDPServer) cleanupInactiveClients() {
	ticker := time.NewTicker(inactivityPeriod)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for key, client := range s.clients {
			if now.Sub(client.lastSeen) > inactivityPeriod {
				log.Printf("Client timed out due to inactivity: %s", client.addr.String())
				delete(s.clients, key)
			}
		}
		s.mu.Unlock()
	}
}

func main() {
	port := "3001"
	server, err := NewUDPServer(port)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %v", err)
	}
	defer server.conn.Close()
	log.Printf("UDP chat server started on :%s", port)
	server.Run()
}
