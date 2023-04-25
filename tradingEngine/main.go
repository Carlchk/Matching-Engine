package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"main/wss"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PriceType int
type OrderSide int

var pairs *string
var priceDigit *int
var quantityDigit *int

const (
	PriceTypeLimit PriceType = 0

	OrderSideBuy  OrderSide = 0
	OrderSideSell OrderSide = 1
)

var sendMsg chan []byte
var tradingServices *TradePair
var recentTrade []interface{}

var web *gin.Engine

func init() {
	pairEnv := os.Getenv("pairs")
	priceDigitEnv := os.Getenv("priceDigit")
	quantityDigitEnv := os.Getenv("quantityDigit")

	// Case: no env
	if len(pairEnv) == 0 {
		pairEnv = "tradingServices"
		pairs = &pairEnv

		priceDigit = new(int)
		*priceDigit = 2

		quantityDigit = new(int)
		*quantityDigit = 4

	} else {
		// Case: With env
		pairs = &pairEnv
		if len(priceDigitEnv) != 0 {
			i64, _ := strconv.Atoi(priceDigitEnv)
			priceDigit = &i64
		}

		if len(quantityDigitEnv) != 0 {
			i64, _ := strconv.Atoi(quantityDigitEnv)
			quantityDigit = &i64
		}
	}
}

func main() {
	port := flag.String("port", "8080", "port")
	flag.Parse()
	gin.SetMode(gin.DebugMode)

	tradingServices = NewTradePair(*pairs, *priceDigit, *quantityDigit)
	recentTrade = make([]interface{}, 0)

	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	startWebServices(*port)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST,HEAD,PATCH, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func startWebServices(port string) {
	web = gin.New()
	web.Use(CORSMiddleware())
	sendMsg = make(chan []byte, 100)

	go pushDepth()
	go watchTradeLog()
	go MQStart()

	web.GET("/api/depth", depth)
	web.GET("/api/trade_log", trade_log)
	web.POST("/api/cancel_order", cancelOrder)

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
	a := tradingServices.GetAskDepth(limitInt)
	b := tradingServices.GetBidDepth(limitInt)

	c.JSON(200, gin.H{
		"ask": a,
		"bid": b,
	})
}

func pushDepth() {
	for {
		ask := tradingServices.GetAskDepth(10)
		bid := tradingServices.GetBidDepth(10)

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

func cancelOrder(c *gin.Context) {
	type args struct {
		OrderId string `json:"order_id"`
	}

	var param args
	c.BindJSON(&param)

	if param.OrderId == "" {
		c.Abort()
		return
	}

	// Signal the tradingEngine to cancel the order
	tradingServices.CancelOrder(param.OrderId)

	go sendMessage("cancel_order", param)

	c.JSON(200, gin.H{
		"ok": true,
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
		case log, ok := <-tradingServices.ChTradeResult:
			if ok {
				relog := gin.H{
					"TradePrice":    tradingServices.Price2String(log.TradePrice),
					"TradeAmount":   tradingServices.Price2String(log.TradeAmount),
					"TradeQuantity": tradingServices.Qty2String(log.TradeQuantity),
					"TradeTime":     log.TradeTime,
					"AskOrderId":    log.AskOrderId,
					"BidOrderId":    log.BidOrderId,
				}
				sendMessage("trade", relog)

				relogJSON, err := json.Marshal(relog)
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				jsonStr := string(relogJSON)

				pushToOutputqueue(jsonStr)

				if len(recentTrade) >= 10 {
					recentTrade = recentTrade[1:]
				}
				recentTrade = append(recentTrade, relog)

				//latest price
				sendMessage("latest_price", gin.H{
					"latest_price": tradingServices.Price2String(log.TradePrice),
				})

			}
		case cancelOrderId := <-tradingServices.ChCancelResult:
			sendMessage("cancel_order", gin.H{
				"OrderId": cancelOrderId,
			})
		default:
			time.Sleep(time.Duration(100) * time.Millisecond)
		}

	}
}
