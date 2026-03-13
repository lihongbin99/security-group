package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"time"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	IP      string `json:"ip,omitempty"`
}

func main() {
	url := flag.String("url", "", "server URL (e.g. https://example.com/api/update)")
	username := flag.String("username", "", "username")
	password := flag.String("password", "", "password")
	interval := flag.Duration("interval", time.Minute, "check interval")
	flag.Parse()

	if *url == "" || *username == "" || *password == "" {
		flag.Usage()
		return
	}

	log.Printf("starting client, url=%s user=%s interval=%s", *url, *username, *interval)

	update(*url, *username, *password)

	ticker := time.NewTicker(*interval)
	for range ticker.C {
		update(*url, *username, *password)
	}
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

func update(url, username, password string) {
	body, _ := json.Marshal(map[string]string{
		"username":  username,
		"password":  password,
		"local_ip":  getLocalIP(),
	})

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("decode response failed: %v", err)
		return
	}

	if result.Code == 0 {
		log.Printf("success: %s (ip: %s)", result.Message, result.IP)
	} else {
		log.Printf("error: %s", result.Message)
	}
}
