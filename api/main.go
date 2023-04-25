package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
)

func main() {
	app := fiber.New()
	app.Use(cors.New())
	createOutputQueue()

	// POST /new_order
	app.Post("/new_order", func(c *fiber.Ctx) error {
		payload := struct {
			Pairs      string `json:"pairs"`
			OrderId    string `json:"order_id"`
			OrderType  string `json:"order_type"`
			PriceType  string `json:"price_type"`
			Price      string `json:"price"`
			Quantity   string `json:"quantity"`
			Amount     string `json:"amount"`
			CreateTime int64  `json:"create_time"`
		}{}

		if err := c.BodyParser(&payload); err != nil {
			return err
		}

		ts := time.Now().UnixNano()
		orderId := uuid.NewString()

		// payload.OrderId = orderId
		payload.CreateTime = ts

		if strings.ToLower(payload.OrderType) == "ask" {
			payload.OrderId = fmt.Sprintf("a-%s", orderId)
		} else {
			payload.OrderId = fmt.Sprintf("b-%s", orderId)
		}

		// Parse payload to json string
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return c.SendString(err.Error())
		}

		publishToTradingEngine(string(jsonPayload), fmt.Sprintf("%s-key", payload.Pairs))

		return c.JSON(payload)
	})

	log.Fatal(app.Listen(":3001"))
}
