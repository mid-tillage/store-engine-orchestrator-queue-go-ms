```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

type Customer struct {
	Code int64
}

type Product struct {
	Code     int64
	Quantity int8
}

type Payment struct {
	Card         Card
	Billing      Billing
	Transaction  Transaction
	Verification Verification
}

type Card struct {
	Pan                   int64
	ExpirationDate        time.Time
	CardholderName        string
	CardVerificationToken string
}

type Billing struct {
	Address string
	ZipCode string
	Country string
}

type Transaction struct {
	Amount            float64
	CurrencyCode      string
	PaymentMethodCode string
	OrderNumber       int64
}

type Shipping struct {
	Geolocation GeoLocation
	Address     AddressCl
	Receiver    Receiver
}

type GeoLocation struct {
	Latitude  float64
	Longitude float64
}

type AddressCl struct {
	RegionCode  int8
	RegionName  string
	ComunaCode  int16
	ComunaName  string
	CalleName   string
	CalleNumber string
	Comments    string
}

type Receiver struct {
	Name string
}

type Verification struct {
	AuthorizationCode string
	TransactionStatus string
	TransactionId     string
	Timestamp         time.Time
}

type Sale struct {
	IdSale   int64
	Customer Customer
	Products []Product
	Payments []Payment
	Shipping Shipping
}

func main() {
	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "mypassword",     // No password set
		DB:       0,                // Use default DB
	})

	// Initialize Gin router
	router := gin.Default()
	router.POST("/sale", postSale)
	router.GET("/sale", getSale)
	go router.Run(":3050")

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

	c.JSON(http.StatusCreated, gin.H{"message": "Sale successfully stored"})
}

func storeSale(sale *Sale) error {
	// Marshal the sale to JSON
	requestBody, err := json.Marshal(sale)
	if err != nil {
		return fmt.Errorf("failed to marshal sale to JSON: %v", err)
	}

	// Store the sale in Redis
	if err := rdb.RPush(context.Background(), "sales_queue", requestBody).Err(); err != nil {
		return fmt.Errorf("failed to push sale to Redis queue: %v", err)
	}

	return nil
}

func getSale(c *gin.Context) {
	// Pop the first sale from the queue
	saleJSON, err := rdb.LPop(context.Background(), "sales_queue").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve sale from the queue"})
		return
	}

	// Unmarshal the sale from JSON
	var sale Sale
	if err := json.Unmarshal([]byte(saleJSON), &sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal sale"})
		return
	}

	c.JSON(http.StatusOK, sale)
}

```