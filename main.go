package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
)

// Response структура ответа JSON
type Response struct {
	Result float64 `json:"result"`
}

// RTPService хранит состояние нашего сервиса
type RTPService struct {
	sync.Mutex // мьютекс для защиты состояния от гонки данных
	rtp        float64
	balance    float64 // текущий баланс "долга/профицита" сервиса

	// Параметры для генерации
	mLow     float64
	mHigh    float64
	pNeutral float64 // базовая вероятность "выигрыша"
	k        float64 // коэффициент чувствительности баланса
}

// NewRTPService создает и инициализирует новый сервис
func NewRTPService(rtp float64) *RTPService {
	// Базовая вероятность выигрыша (pNeutral) рассчитывается так,
	// чтобы математическое ожидание мультипликатора при нулевом балансе было равно rtp.
	// E(m) = p * mHigh + (1-p) * mLow = rtp
	// p * (mHigh - mLow) = rtp - mLow
	// p = (rtp - mLow) / (mHigh - mLow)
	mLow := 1.0
	mHigh := 10000.0
	pNeutral := (rtp - mLow) / (mHigh - mLow)

	//log.Printf("Target RTP=%.4f, mLow=%.1f, mHigh=%.1f", rtp, mLow, mHigh)
	//log.Printf("Calculated neutral probability (pNeutral): %.6f", pNeutral)

	return &RTPService{
		rtp:      rtp,
		balance:  0.0,
		mLow:     mLow,
		mHigh:    mHigh,
		pNeutral: pNeutral,
		k:        0.0001, // Коэффициент можно подбирать для более плавной/резкой коррекции
	}
}

// GetMultiplier генерирует новый мультипликатор, корректируя вероятность на основе баланса
func (s *RTPService) GetMultiplier() float64 {
	s.Lock()
	defer s.Unlock()

	// Корректируем вероятность в зависимости от баланса
	// Если баланс положительный (сервис "должен" игрокам), повышаем шанс выигрыша
	// Если баланс отрицательный (сервис "переплатил"), понижаем
	pAdjusted := s.pNeutral + s.balance*s.k

	// Ограничиваем вероятность в пределах [0, 1]
	if pAdjusted < 0 {
		pAdjusted = 0
	} else if pAdjusted > 1 {
		pAdjusted = 1
	}

	var multiplier float64
	if rand.Float64() < pAdjusted {
		multiplier = s.mHigh
	} else {
		multiplier = s.mLow
	}

	// Обновляем баланс:
	// Прибавляем то, что должны были вернуть (rtp)
	// Вычитаем то, что вернули по факту (multiplier)
	s.balance += s.rtp - multiplier

	// Ограничиваем рост баланса, чтобы избежать слишком больших колебаний
	// Этот шаг опционален, но помогает стабилизировать систему
	maxBalance := s.mHigh * 10
	if s.balance > maxBalance {
		s.balance = maxBalance
	} else if s.balance < -maxBalance {
		s.balance = -maxBalance
	}

	return multiplier
}

func main() {
	var rtp float64
	flag.Float64Var(&rtp, "rtp", 0.0, "target RTP value in (0,1.0]")
	flag.Parse()

	if rtp <= 0.0 || rtp > 1.0 {
		fmt.Println("Error: -rtp must be a positive value <=1")
		os.Exit(1)
	}

	service := NewRTPService(rtp)

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		multiplier := service.GetMultiplier()

		resp := Response{Result: multiplier}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	addr := ":64333"
	log.Printf("Starting stateful RTP service on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
