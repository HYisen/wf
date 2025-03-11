package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
)

func main() {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/events", nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Status)
	body := resp.Body
	//goland:noinspection GoUnhandledErrorResult
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	log.Println("END")
}
