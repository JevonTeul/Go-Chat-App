package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:3001")
	if err != nil {
		log.Fatalf("Failed to resolve server: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer conn.Close()

	fmt.Print("Enter username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := scanner.Text()
	conn.Write([]byte(username))

	go func() {
		buf := make([]byte, 2048)
		var lastMsg time.Time
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Printf("Read error: %v", err)
				return
			}
			msg := strings.TrimSpace(string(buf[:n]))
			now := time.Now()
			if !lastMsg.IsZero() {
				delay := now.Sub(lastMsg).Milliseconds()
				fmt.Printf("[Delay: %dms] %s\n", delay, msg)
			} else {
				fmt.Println(msg)
			}
			lastMsg = now
		}
	}()

	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		if _, err := conn.Write([]byte(text)); err != nil {
			log.Printf("Send failed: %v", err)
			break
		}
		if text == "/quit" || text == "bye" {
			fmt.Println("Disconnecting...")
			break
		}
	}
}
