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
)

func main() {
	// Парсим флаг rtp
	flag.Float64Var(&rtp, "rtp", 0.0, "target RTP value in (0,1]")
	flag.Parse()

	if rtp <= 0.0 || rtp > 1.0 {
		fmt.Println("Error: -rtp must be in (0,1]")
		os.Exit(1)
	}

	// Предварительно вычисляем вероятность p для H=101
	H := 101.0
	p = (9999.0 * rtp) / (H * (H - 1))

	http.HandleFunc("/get", handleGet)

	addr := ":64333"
	log.Printf("Starting RTP service on %s with target RTP=%.4f (p=%.6f)\n", addr, rtp, p)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	u := rand.Float64()
	var result float64
	if u < p {
		result = 101.0
	} else {
		result = 1.0
	}

	resp := Response{Result: result}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
