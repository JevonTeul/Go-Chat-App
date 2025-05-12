package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:3000")
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	fmt.Print("Enter your username: ")
	usernameScanner := bufio.NewScanner(os.Stdin)
	usernameScanner.Scan()
	username := usernameScanner.Text()

	go func() {
		var lastMsgTime time.Time
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			now := time.Now()
			if !lastMsgTime.IsZero() {
				delay := now.Sub(lastMsgTime).Milliseconds()
				fmt.Printf("[Delay: %dms] %s\n", delay, scanner.Text())
			} else {
				fmt.Println(scanner.Text())
			}
			lastMsgTime = now
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading from server: %v", err)
		}
		os.Exit(0)
	}()

	inputScanner := bufio.NewScanner(os.Stdin)
	for inputScanner.Scan() {
		text := inputScanner.Text()
		if text == "" {
			continue
		}
		_, err := fmt.Fprintf(conn, "[%s] %s\n", username, text)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			break
		}
	}
	if err := inputScanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}
}
