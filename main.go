package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var apiKeys = []string{
	"CG-TxU95gTnKpbSU69rXBPDRZfn",
	"CG-MuKoMVL1JZ8Mv4BhcMYPEF6M",
	"CG-qb3LHfPWCdpkbpyU2kDEnAHD",
	"CG-48RBhU9Ht18XsBYKVW6m2tkT",
	"CG-zprA3h6Nog7ZoivMDq6CHtLN",
	"CG-4B7ieXZ6Wdq7mz7PPsJkDXpz",
	"CG-P9gGNPKYsrNH4Lz7ZhLkYGWi",
	"CG-zprA3h6Nog7ZoivMDq6CHtLN",
	"CG-qb3LHfPWCdpkbpyU2kDEnAHD",
	"CG-MuKoMVL1JZ8Mv4BhcMYPEF6M",
}

func main() {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	router := gin.Default()

	router.GET("/inference/:token", func(c *gin.Context) {
		token := c.Param("token")
		price, err := getLastPrice(token)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error: %v", err)
			return
		}
		c.String(http.StatusOK, price)
	})

	router.Run(":8000")
}

func getSimplePrice(token string) (string, error) {
	baseURL := "https://api.coingecko.com/api/v3/simple/price"
	tokenMap := map[string]string{
		"ETH": "ethereum",
		"SOL": "solana",
		"BTC": "bitcoin",
		"BNB": "binancecoin",
		"ARB": "arbitrum",
	}

	// Randomly select an API key
	selectedKey := apiKeys[rand.Intn(len(apiKeys))]

	headers := map[string]string{
		"accept":             "application/json",
		"x-cg-demo-api-key": selectedKey,
	}

	token = strings.ToUpper(token)
	tokenID, ok := tokenMap[token]
	if !ok {
		tokenID = strings.ToLower(token)
	}

	url := fmt.Sprintf("%s?ids=%s&vs_currencies=usd", baseURL, tokenID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code %d", resp.StatusCode)
	}

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	price, ok := result[tokenID]["usd"]
	if !ok {
		return "", fmt.Errorf("price not found for token %s", tokenID)
	}

	return fmt.Sprintf("%.2f", price), nil
}

func getLastPrice(token string) (string, error) {
	currentPriceStr, err := getSimplePrice(token)
	if err != nil {
		return "", err
	}

	currentPrice, err := strconv.ParseFloat(currentPriceStr, 64)
	if err != nil {
		return "", err
	}

	// Randomly adjust price by ±0.8%
	adjustmentFactor := rand.Float64()*0.016 + 0.992
	adjustedPrice := currentPrice * adjustmentFactor

	return fmt.Sprintf("%.2f", adjustedPrice), nil
}
