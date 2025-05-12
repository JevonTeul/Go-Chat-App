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
		log.Fatalf("Failed to resolve server address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP server: %v", err)
	}
	defer conn.Close()

	var lastMsgTime time.Time

	// Goroutine to listen for incoming messages
	go func() {
		buf := make([]byte, 2048)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Printf("Error reading from server: %v", err)
				return
			}
			message := strings.TrimSpace(string(buf[:n]))

			// Calculate delay between messages
			now := time.Now()
			if !lastMsgTime.IsZero() {
				delay := now.Sub(lastMsgTime).Milliseconds()
				fmt.Printf("[Delay: %dms] %s\n", delay, message)
			} else {
				fmt.Println(message)
			}

			lastMsgTime = now
		}
	}()

	// Scanner for reading user input
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}

		// Send the message to the server
		_, err := conn.Write([]byte(text))
		if err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}

		if text == "/quit" || text == "bye" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}
