package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
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

type envConfig struct {
	APIKey string `json:"api_key"`
	RPC    string `json:"rpc"`
}

func main() {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	cfg := &envConfig{
		APIKey: os.Getenv("UPSHOT_APIKEY"),
		RPC:    os.Getenv("RPC"),
	}

	fmt.Println("UPSHOT_APIKEY: ", cfg.APIKey)
	fmt.Println("RPC: ", cfg.RPC)

	router := gin.Default()

	router.GET("/inference/:token", func(c *gin.Context) {
		token := c.Param("token")
		if token == "MEME" {
			handleMemeRequest(c, cfg)
			return
		}

		price, err := getAdjustedPrice(token)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error: %v", err)
			return
		}

		c.String(http.StatusOK, price)
	})

	router.Run(":8000")
}

func getAdjustedPrice(token string) (string, error) {
	price, err := getSimplePrice(token)
	if err != nil {
		return "", err
	}

	currentPrice, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "", err
	}

	// Randomly adjust price by ±0.8%
	adjustmentFactor := rand.Float64()*0.016 + 0.992
	adjustedPrice := currentPrice * adjustmentFactor

	return fmt.Sprintf("%.2f", adjustedPrice), nil
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

func handleMemeRequest(c *gin.Context, cfg *envConfig) {
	if cfg.APIKey == "" {
		c.String(http.StatusBadRequest, "need API key")
		return
	}

	if cfg.RPC == "" {
		c.String(http.StatusInternalServerError, "Invalid RPC URL")
		return
	}

	lb, err := getLatestBlock(cfg.RPC)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching latest block: %v", err)
		return
	}

	meme, err := getMemeOracleData(lb, cfg.APIKey)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching meme data: %v", err)
		return
	}

	mp, err := getMemePrice(meme.Data.Platform, meme.Data.Address)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error fetching meme price: %v", err)
		return
	}

	mpf, _ := strconv.ParseFloat(mp, 64)
	adjustedPrice := random(mpf)

	c.String(http.StatusOK, fmt.Sprintf("%.6f", adjustedPrice))
}

func getMemePrice(network, memeAddress string) (string, error) {
	url := fmt.Sprintf("https://api.geckoterminal.com/api/v2/simple/networks/%s/token_price/%s", network, memeAddress)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var res tokenPriceResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	price, ok := res.Data.Attributes.TokenPrices[memeAddress]
	if !ok {
		return "", fmt.Errorf("price not found for address %s", memeAddress)
	}

	return price, nil
}

func getLatestBlock(rpc string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/status", rpc), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create new request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response latestBlockResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Result.SyncInfo.LatestBlockHeight, nil
}

func getMemeOracleData(blockHeight, apiKey string) (*memeOracleResponse, error) {
	url := fmt.Sprintf("https://api.upshot.xyz/v2/allora/tokens-oracle/token/%s", blockHeight)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res memeOracleResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &res, nil
}

func random(price float64) float64 {
	randomPercent := rand.Float64()*6 - 3
	priceChange := price * (randomPercent / 100)
	return price + priceChange
}

type tokenPriceResponse struct {
	Data struct {
		Attributes struct {
			TokenPrices map[string]string `json:"token_prices"`
		} `json:"attributes"`
	} `json:"data"`
}

type latestBlockResponse struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

type memeOracleResponse struct {
	RequestID string `json:"request_id"`
	Status    bool   `json:"status"`
	Data      struct {
		TokenID     string `json:"token_id"`
		TokenSymbol string `json:"token_symbol"`
		Platform    string `json:"platform"`
		Address     string `json:"address"`
	} `json:"data"`
}
