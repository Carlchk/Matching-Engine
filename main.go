package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"main/wss"

	_ "net/http/pprof"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
)

type PriceType int
type OrderSide int

const (
	PriceTypeLimit          PriceType = 0
	PriceTypeMarket         PriceType = 1
	PriceTypeMarketQuantity PriceType = 2
	PriceTypeMarketAmount   PriceType = 3

	OrderSideBuy  OrderSide = 0
	OrderSideSell OrderSide = 1
)

var sendMsg chan []byte
var btcusdt *TradePair
var recentTrade []interface{}

var web *gin.Engine

func main() {
	port := flag.String("port", "8080", "port")
	flag.Parse()
	gin.SetMode(gin.DebugMode)

	btcusdt = NewTradePair("BTC_USDT", 2, 4)
	recentTrade = make([]interface{}, 0)

	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	startWeb(*port)
}

func startWeb(port string) {
	web = gin.New()
	web.LoadHTMLGlob("./webapp/*.html")
	web.StaticFS("/webapp", http.Dir("./webapp"))

	sendMsg = make(chan []byte, 100)

	go pushDepth()
	go watchTradeLog()

	web.GET("/api/depth", depth)
	web.GET("/api/trade_log", trade_log)
	web.POST("/api/new_order", newOrder)
	// web.POST("/api/cancel_order", cancelOrder)
	// web.GET("/api/test_rand", testOrder)

	web.GET("/", func(c *gin.Context) {
		c.HTML(200, "demo.html", nil)
	})

	web.GET("/preview", func(c *gin.Context) {
		c.HTML(200, "main.html", nil)
	})

	//websocket
	{
		wss.HHub = wss.NewHub()
		go wss.HHub.Run()
		go func() {
			for {
				select {
				case data := <-sendMsg:
					wss.HHub.Send(data)
				default:
					time.Sleep(time.Duration(100) * time.Millisecond)
				}
			}
		}()

		web.GET("/ws", wss.ServeWs)
		web.GET("/pong", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}

	web.Run(":" + port)
}

func depth(c *gin.Context) {
	limit := c.Query("limit")
	limitInt, _ := strconv.Atoi(limit)
	if limitInt <= 0 || limitInt > 100 {
		limitInt = 10
	}
	a := btcusdt.GetAskDepth(limitInt)
	b := btcusdt.GetBidDepth(limitInt)

	c.JSON(200, gin.H{
		"ask": a,
		"bid": b,
	})
}

func pushDepth() {
	for {
		ask := btcusdt.GetAskDepth(10)
		bid := btcusdt.GetBidDepth(10)

		sendMessage("depth", gin.H{
			"ask": ask,
			"bid": bid,
		})

		time.Sleep(time.Duration(150) * time.Millisecond)
	}
}

func trade_log(c *gin.Context) {
	c.JSON(200, gin.H{
		"ok": true,
		"data": gin.H{
			"trade_log": recentTrade,
		},
	})
}

func newOrder(c *gin.Context) {
	type args struct {
		OrderId   string `json:"order_id"`
		OrderType string `json:"order_type"`
		PriceType string `json:"price_type"`
		Price     string `json:"price"`
		Quantity  string `json:"quantity"`
		Amount    string `json:"amount"`
	}

	var param args
	c.BindJSON(&param)

	orderId := uuid.NewString()
	param.OrderId = orderId

	price := string2decimal(param.Price)
	quantity := string2decimal(param.Quantity)

	var pt PriceType

	pt = PriceTypeLimit
	param.Amount = "0"
	if price.Cmp(decimal.NewFromFloat(100000000)) > 0 || price.Cmp(decimal.Zero) < 0 {
		c.JSON(200, gin.H{
			"ok":    false,
			"error": "Price must be > 0 and < 100000000",
		})
		return
	}
	if quantity.Cmp(decimal.NewFromFloat(100000000)) > 0 || quantity.Cmp(decimal.Zero) <= 0 {
		c.JSON(200, gin.H{
			"ok":    false,
			"error": "Quantity must be > 0 and < 100000000",
		})
		return
	}

	if strings.ToLower(param.OrderType) == "ask" {
		param.OrderId = fmt.Sprintf("a-%s", orderId)
		item := NewAskItem(pt, param.OrderId, string2decimal(param.Price), string2decimal(param.Quantity), string2decimal(param.Amount), time.Now().UnixNano())
		btcusdt.ChNewOrder <- item

	} else {
		param.OrderId = fmt.Sprintf("b-%s", orderId)
		item := NewBidItem(pt, param.OrderId, string2decimal(param.Price), string2decimal(param.Quantity), string2decimal(param.Amount), time.Now().UnixNano())
		btcusdt.ChNewOrder <- item
	}

	go sendMessage("new_order", param)

	c.JSON(200, gin.H{
		"ok": true,
		"data": gin.H{
			"ask_len": btcusdt.AskLen(),
			"bid_len": btcusdt.BidLen(),
		},
	})
}

func sendMessage(tag string, data interface{}) {
	msg := gin.H{
		"tag":  tag,
		"data": data,
	}
	msgByte, _ := json.Marshal(msg)
	sendMsg <- []byte(msgByte)
}

func watchTradeLog() {
	for {
		select {
		case log, ok := <-btcusdt.ChTradeResult:
			if ok {
				//

				relog := gin.H{
					"TradePrice":    btcusdt.Price2String(log.TradePrice),
					"TradeAmount":   btcusdt.Price2String(log.TradeAmount),
					"TradeQuantity": btcusdt.Qty2String(log.TradeQuantity),
					"TradeTime":     log.TradeTime,
					"AskOrderId":    log.AskOrderId,
					"BidOrderId":    log.BidOrderId,
				}
				sendMessage("trade", relog)

				if len(recentTrade) >= 10 {
					recentTrade = recentTrade[1:]
				}
				recentTrade = append(recentTrade, relog)

				//latest price
				sendMessage("latest_price", gin.H{
					"latest_price": btcusdt.Price2String(log.TradePrice),
				})

			}
		case cancelOrderId := <-btcusdt.ChCancelResult:
			sendMessage("cancel_order", gin.H{
				"OrderId": cancelOrderId,
			})
		default:
			time.Sleep(time.Duration(100) * time.Millisecond)
		}

	}
}
