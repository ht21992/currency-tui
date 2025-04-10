package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/go-redis/redis"
)

type ExchangeRateResponse struct {
	Rates map[string]float64 `json:"rates"`
}

var supportedCurrencies map[string]bool

func sortCurrencies(currencies, favorites []string) []string {
	// Use a map for fast favorite lookup
	favMap := make(map[string]bool)
	for _, fav := range favorites {
		favMap[fav] = true
	}

	// Split into favorites and the rest
	var regular []string
	for _, currency := range currencies {
		if !favMap[currency] {
			regular = append(regular, currency)
		}
	}

	// Sort non-favorites
	sort.Strings(regular)

	// Combine favorites + sorted regular
	currencies = append(favorites, regular...)

	return currencies
}

func getCurrenciesFromRedis(key string) ([]string, error) {
	var currencies []string

	val, err := client.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist
			return nil, nil
		}
		// Some other Redis error
		return nil, err
	}

	// Try to unmarshal the JSON value
	err = json.Unmarshal([]byte(val), &currencies)
	if err != nil {
		return nil, err
	}

	return currencies, nil
}

func fetchSupportedCurrencies(base string) ([]string, error) {
	var currencies []string
	favorites := []string{"USD", "EUR", "GBP"}

	// Check Redis first
	currencies, err := getCurrenciesFromRedis("supported_currencies")
	if err != nil {
		return nil, err
	}
	if len(currencies) > 0 {
		infoColor.Println("Loaded currencies from Redis")
		return sortCurrencies(currencies, favorites), nil
	}

	infoColor.Println("Fetching currencies from API...")
	url := fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", base)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch currency list: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data ExchangeRateResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exchange data: %v", err)
	}

	currencies = make([]string, 0, len(data.Rates))
	for code := range data.Rates {
		currencies = append(currencies, code)
	}

	// Convert slice to JSON
	currenciesJsonData, err := json.Marshal(currencies)
	if err != nil {
		return nil, err
	}
	// Set a key-value pair
	err = client.Set("supported_currencies", currenciesJsonData, 0).Err()
	if err != nil {
		panic(err)
	}
	currencies = sortCurrencies(currencies, favorites)
	return currencies, nil
}

func getExchangeRate(from, to string) (float64, error) {
	url := fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", from)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch rates: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data ExchangeRateResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, fmt.Errorf("failed to parse exchange data: %v", err)
	}

	// Store supported currencies
	supportedCurrencies = make(map[string]bool)
	for currency := range data.Rates {
		supportedCurrencies[currency] = true
	}

	rate, ok := data.Rates[to]
	if !ok {
		return 0, fmt.Errorf("currency %s not supported", to)
	}

	return rate, nil
}
