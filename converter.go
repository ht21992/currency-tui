package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ExchangeRateResponse struct {
	Rates map[string]float64 `json:"rates"`
}

var supportedCurrencies map[string]bool

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

func isValidCurrency(code string) bool {
	if supportedCurrencies == nil {
		_, _ = getExchangeRate("USD", "EUR") // preload map
	}
	code = strings.ToUpper(code)
	return supportedCurrencies[code]
}
