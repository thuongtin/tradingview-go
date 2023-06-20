package binance

import (

	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/gin-gonic/gin"
	"github.com/nathan-tw/tradingview-go/src/webhook"

)

var (
	apiKey    string = os.Getenv("BINANCE_API_KEY")
	apiSecret string = os.Getenv("BINANCE_API_SECRET")
	QuantityPrecisions = make(map[string]string)
)

type ExchangeInfoDto struct {
	Timezone    string `json:"timezone"`
	ServerTime  int64  `json:"serverTime"`
	FuturesType string `json:"futuresType"`
	RateLimits  []struct {
		RateLimitType string `json:"rateLimitType"`
		Interval      string `json:"interval"`
		IntervalNum   int    `json:"intervalNum"`
		Limit         int    `json:"limit"`
	} `json:"rateLimits"`
	ExchangeFilters []any `json:"exchangeFilters"`
	Assets          []struct {
		Asset             string `json:"asset"`
		MarginAvailable   bool   `json:"marginAvailable"`
		AutoAssetExchange string `json:"autoAssetExchange"`
	} `json:"assets"`
	Symbols []Symbol `json:"symbols"`
}

type Symbol struct {
	Symbol                string `json:"symbol"`
	Pair                  string `json:"pair"`
	ContractType          string `json:"contractType"`
	DeliveryDate          int64  `json:"deliveryDate"`
	OnboardDate           int64  `json:"onboardDate"`
	Status                string `json:"status"`
	MaintMarginPercent    string `json:"maintMarginPercent"`
	RequiredMarginPercent string `json:"requiredMarginPercent"`
	BaseAsset             string `json:"baseAsset"`
	QuoteAsset            string `json:"quoteAsset"`
	MarginAsset           string `json:"marginAsset"`
	PricePrecision        int    `json:"pricePrecision"`
	QuantityPrecision     int    `json:"quantityPrecision"`
	BaseAssetPrecision    int    `json:"baseAssetPrecision"`
	QuotePrecision        int    `json:"quotePrecision"`
	UnderlyingType        string `json:"underlyingType"`
	UnderlyingSubType     []any  `json:"underlyingSubType"`
	SettlePlan            int    `json:"settlePlan"`
	TriggerProtect        string `json:"triggerProtect"`
	LiquidationFee        string `json:"liquidationFee"`
	MarketTakeBound       string `json:"marketTakeBound"`
	MaxMoveOrderLimit     int    `json:"maxMoveOrderLimit"`
	Filters               []struct {
		MinPrice          string `json:"minPrice,omitempty"`
		MaxPrice          string `json:"maxPrice,omitempty"`
		FilterType        string `json:"filterType"`
		TickSize          string `json:"tickSize,omitempty"`
		StepSize          string `json:"stepSize,omitempty"`
		MaxQty            string `json:"maxQty,omitempty"`
		MinQty            string `json:"minQty,omitempty"`
		Limit             int    `json:"limit,omitempty"`
		Notional          string `json:"notional,omitempty"`
		MultiplierDown    string `json:"multiplierDown,omitempty"`
		MultiplierUp      string `json:"multiplierUp,omitempty"`
		MultiplierDecimal string `json:"multiplierDecimal,omitempty"`
	} `json:"filters"`
	OrderTypes  []string `json:"orderTypes"`
	TimeInForce []string `json:"timeInForce"`
}

func init() {
	// Tạo một HTTP client mới
	client := http.Client{}

	// Tạo một GET request đến URL
	req, err := http.NewRequest("GET", "https://fapi.binance.com/fapi/v1/exchangeInfo", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Gửi request và nhận phản hồi từ server
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Đọc dữ liệu từ phản hồi
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var result ExchangeInfoDto
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatal(err)
	}
	for _, symbol := range result.Symbols {
		QuantityPrecisions[symbol.Symbol] = fmt.Sprintf("%%.%df", symbol.QuantityPrecision)
	}
}

func HandleFuturesStrategy(c *gin.Context) {
	if runtime.GOOS == "darwin" {
		futures.UseTestnet = true
	}

	jsonData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		panic(err)
	}
	alert := new(webhook.TradingviewAlert)
	err = json.Unmarshal(jsonData, alert)
	if err != nil {
		panic(err)
	}
	if ok := webhook.ValidatePassPhrase(alert); !ok {
		c.String(http.StatusBadRequest, "wrong passphrase")
		return
	}

	side := strings.ToUpper(alert.Strategy.OrderAction)
	

	symbol := alert.Ticker
	quantity := fmt.Sprintf(QuantityPrecisions[symbol], alert.Strategy.OrderContracts)
	//switch symbol {
	//case "LINKUSDT":
	//	quantity = fmt.Sprintf("%.2f", alert.Strategy.OrderContracts)
	//case "SOLUSDT":
	//	quantity = fmt.Sprintf("%.0f", alert.Strategy.OrderContracts)
	//}
	fmt.Printf("%s trading side: %v, quantity: %v\n", symbol, side, quantity)

	futuresClient := binance.NewFuturesClient(apiKey, apiSecret)
	order, err := futuresClient.NewCreateOrderService().Symbol(symbol).Side(futures.SideType(side)).Type(futures.OrderTypeMarket).Quantity(quantity).Do(context.Background())
	if err != nil {
		c.String(http.StatusBadRequest, "create futures order fail %v", err)
		return
	}
	fmt.Println(order)
	c.String(http.StatusOK, "create futures order success")
}
