package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
	"golang.org/x/net/html"
)

type Kline struct {
	OpenTime  time.Time
	CloseTime time.Time
	Interval  string
	Symbol    string
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	Closed    bool
}

type envConfig struct {
	RPC    string `json:"rpc"`
	APIKey string `json:"api_key"`
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

func main() {
	cfg := &envConfig{
		APIKey: os.Getenv("UPSHOT_APIKEY"),
		RPC:    os.Getenv("RPC"),
	}

	fmt.Println("UPSHOT_APIKEY: ", cfg.APIKey)
	fmt.Println("RPC: ", cfg.RPC)

	router := gin.Default()

	router.GET("/inference/:tokenorblockheight", func(c *gin.Context) {
		param := c.Param("tokenorblockheight")
		var namecoin string

		if isNumeric(param) {
			// Handle block height
			blockHeight := param
			meme, err := getMemeOracleData(blockHeight, cfg.APIKey)
			if err != nil {
				fmt.Println("Error getting token name:", err)
				c.String(http.StatusInternalServerError, "Error retrieving token")
				return
			}
			namecoin = meme.Data.TokenSymbol
		} else {
			// Handle token symbol
			namecoin = param
		}

		price, err := getLastPrice(namecoin)
		if err != nil {
			fmt.Println("Error getting price:", err)
			c.String(http.StatusInternalServerError, "Error retrieving price")
			return
		}

		c.String(http.StatusOK, price)
	})

	router.Run(":8000")
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func getLastPrice(token string) (string, error) {
	// Get token data from CoinGecko
	price, err := getTokenPriceFromCoinGecko(token)
	if err != nil {
		return "", err
	}

	// Adjust price by ±0.8%
	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "", err
	}

	adjustedPrice := adjustPrice(priceFloat)
	return strconv.FormatFloat(adjustedPrice, 'g', -1, 64), nil
}

func getTokenPriceFromCoinGecko(token string) (string, error) {
	baseURL := "https://api.coingecko.com/api/v3/simple/price"
	tokenMap := map[string]string{
		"ETH": "ethereum",
		"SOL": "solana",
		"BTC": "bitcoin",
		"BNB": "binancecoin",
		"ARB": "arbitrum",
	}

	tokenID := tokenMap[token]
	if tokenID == "" {
		tokenID = token
	}

	url := fmt.Sprintf("%s?ids=%s&vs_currencies=usd", baseURL, tokenID)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get price from CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CoinGecko API request failed with status: %d", resp.StatusCode)
	}

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("failed to decode CoinGecko response: %w", err)
	}

	price, ok := data[tokenID]["usd"]
	if !ok {
		return "", fmt.Errorf("price for token %s not found", token)
	}

	return strconv.FormatFloat(price, 'f', 2, 64), nil
}

func adjustPrice(price float64) float64 {
	adjustmentFactor := rand.New(rand.NewSource(uint64(time.Now().UnixNano()))).Float64()*0.016 + 0.992
	return price * adjustmentFactor
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
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	res := &memeOracleResponse{}
	err = json.Unmarshal(body, res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return res, nil
}

func random(price float64) float64 {
	randomPercent := rand.Float64()*6 - 3
	priceChange := price * (randomPercent / 100)
	return price + priceChange
}
