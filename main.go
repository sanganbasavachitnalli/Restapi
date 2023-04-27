package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Transaction struct {
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

type Stats struct {
	Sum   float64 `json:"sum"`
	Avg   float64 `json:"avg"`
	Max   float64 `json:"max"`
	Min   float64 `json:"min"`
	Count int     `json:"count"`
}

type Location struct {
	City string `json:"city"`
}

type StatsCache struct {
	lock        sync.RWMutex
	lastUpdated time.Time
	stats       Stats
	queue       []*Transaction
}

type LocationCache struct {
	lock     sync.RWMutex
	location Location
}

var (
	statsCache    StatsCache
	locationCache LocationCache
)

func main() {
	http.HandleFunc("/transactions", transactionsHandler)
	http.HandleFunc("/statistics", statisticsHandler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/location", locationHandler)
	http.HandleFunc("/location/reset", resetLocationHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func transactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var transaction Transaction
	err := json.NewDecoder(r.Body).Decode(&transaction)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if transaction.Timestamp.After(time.Now().UTC()) {
		http.Error(w, "Transaction timestamp is in the future", http.StatusUnprocessableEntity)
		return
	}

	if time.Since(transaction.Timestamp) > time.Second*60 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	statsCache.lock.Lock()
	defer statsCache.lock.Unlock()

	statsCache.stats.Sum += transaction.Amount
	statsCache.stats.Count++
	if transaction.Amount > statsCache.stats.Max {
		statsCache.stats.Max = transaction.Amount
	}
	if statsCache.stats.Min == 0 || transaction.Amount < statsCache.stats.Min {
		statsCache.stats.Min = transaction.Amount
	}

	statsCache.lastUpdated = time.Now().UTC()

	w.WriteHeader(http.StatusCreated)
}

func statisticsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	locationCache.lock.RLock()
	defer locationCache.lock.RUnlock()

	if locationCache.location.City != "" && locationCache.location.City != "bangalore" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	statsCache.lock.RLock()
	defer statsCache.lock.RUnlock()

	if time.Since(statsCache.lastUpdated) > time.Second*60 {
		fmt.Fprintf(w, "{}")
		return
	}

	stats := statsCache.stats
	stats.Avg = stats.Sum / float64(stats.Count)

	json.NewEncoder(w).Encode(stats)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statsCache.lock.Lock()
	defer statsCache.lock.Unlock()

	statsCache.stats.Sum = 0
	statsCache.stats.Avg = 0
	statsCache.stats.Max = 0
	statsCache.stats.Min = 0
	statsCache.stats.Count = 0
	statsCache.lastUpdated = time.Time{}

	w.WriteHeader(http.StatusNoContent)
}

var currentLocation Location

func locationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loc Location
	err := json.NewDecoder(r.Body).Decode(&loc)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	locationCache.location = loc

	w.WriteHeader(http.StatusNoContent)
}

func resetLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	locationCache.location = Location{}

	w.WriteHeader(http.StatusNoContent)
}
