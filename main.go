package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var rdb *redis.Client

type Customer struct {
	Code int64 `json:"code"`
}

type Product struct {
	Code     int64 `json:"code"`
	Quantity int8  `json:"quantity"`
}

type Payment struct {
	Card         Card         `json:"card"`
	Billing      Billing      `json:"billing"`
	Transaction  Transaction  `json:"transaction"`
	Verification Verification `json:"verification"`
}

type Card struct {
	Pan                   int64     `json:"pan"`
	ExpirationDate        time.Time `json:"expirationDate"`
	CardholderName        string    `json:"cardholderName"`
	CardVerificationToken string    `json:"cardVerificationToken"`
}

type Billing struct {
	Address string `json:"address"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

type Transaction struct {
	Amount            float64 `json:"amount"`
	CurrencyCode      string  `json:"currencyCode"`
	PaymentMethodCode string  `json:"paymentMethodCode"`
	OrderNumber       int64   `json:"orderNumber"`
}

type Shipping struct {
	Geolocation GeoLocation `json:"geolocation"`
	Address     AddressCl   `json:"address"`
	Receiver    Receiver    `json:"receiver"`
}

type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type AddressCl struct {
	RegionCode  int8   `json:"regionCode"`
	RegionName  string `json:"regionName"`
	ComunaCode  int16  `json:"comunaCode"`
	ComunaName  string `json:"comunaName"`
	CalleName   string `json:"calleName"`
	CalleNumber string `json:"calleNumber"`
	Comments    string `json:"comments"`
}

type Receiver struct {
	Name string `json:"name"`
}

type Verification struct {
	AuthorizationCode string    `json:"authorizationCode"`
	TransactionStatus string    `json:"transactionStatus"`
	TransactionId     string    `json:"transactionId"`
	Timestamp         time.Time `json:"timestamp"`
}

type Sale struct {
	IdSale   int64     `json:"idSale"`
	Customer Customer  `json:"customer"`
	Products []Product `json:"products"`
	Payments []Payment `json:"payments"`
	Shipping Shipping  `json:"shipping"`
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("QUEUE_REDIS_IP") + ":" + os.Getenv("QUEUE_REDIS_PORT"),
		Password: os.Getenv("QUEUE_REDIS_PASSWORD"),
		DB:       0,
	})

	// Initialize Gin router
	router := gin.Default()
	router.POST("/sale", postSale)
	go func() {
		if err := router.Run(":" + os.Getenv("STORE_ENGINE_ORCHESTRATOR_QUEUE_SERVER_PORT")); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	fmt.Println("Shutting down...")
}

func postSale(c *gin.Context) {
	var sale Sale
	if err := c.BindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := storeSale(&sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store sale", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Sale successfully processed pushed into the queue"})
}

func storeSale(sale *Sale) error {
	ctx := context.Background()

	requestBody, err := json.Marshal(sale)
	if err != nil {
		return fmt.Errorf("failed to marshal sale to JSON: %v", err)
	}

	if err := rdb.RPush(ctx, "sales_queue", requestBody).Err(); err != nil {
		return fmt.Errorf("failed to push sale to Redis queue: %v", err)
	}

	return nil
}
