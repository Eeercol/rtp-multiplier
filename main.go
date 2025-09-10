package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
)

// Response структура ответа JSON
type Response struct {
	Result float64 `json:"result"`
}

var (
	rtp float64
	p   float64
	H   float64 = 100.0 // фиксируем большой выигрыш
)

func main() {
	// Парсим флаг rtp
	flag.Float64Var(&rtp, "rtp", 0.0, "target RTP value in (0,1]")
	flag.Parse()

	if rtp <= 0.0 || rtp > 1.0 {
		fmt.Println("Error: -rtp must be in (0,1]")
		os.Exit(1)
	}

	// Вероятность выпадения выигрыша
	p = rtp / H

	http.HandleFunc("/get", handleGet)

	addr := ":64333"
	log.Printf("Starting RTP service on %s with target RTP=%.4f (H=%.1f, p=%.6f)\n", addr, rtp, H, p)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	u := rand.Float64()
	var result float64
	if u < p {
		result = H
	} else {
		result = 0
	}

	resp := Response{Result: result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
