package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type EncodedData struct {
	EncodedString string `json:"encoded_string"`
}

func EncodeData(data interface{}) (EncodedData, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return EncodedData{}, fmt.Errorf("error encoding data: %v", err)
	}

	encodedString := "a." + base64.StdEncoding.EncodeToString(jsonData)

	return EncodedData{
		EncodedString: encodedString,
	}, nil
}

type Identity struct {
	AccountNumber     string `json:"account_number"`
	LastName          string `json:"last_name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	AccountID         string `json:"account_id"`
	UserID            string `json:"user_id"`
	OrgID             string `json:"org_id"`
	Username          string `json:"username"`
}

func createIdentity() Identity {
	i := Identity{
		AccountNumber:     fmt.Sprint(rand.Intn(1000)),
		LastName:          "Bar",
		PreferredUsername: "Foo",
		GivenName:         "Foo",
		AccountID:         fmt.Sprint(rand.Intn(100000)),
		UserID:            fmt.Sprint(rand.Intn(100000)),
		OrgID:             fmt.Sprint(rand.Intn(1000)),
		Username:          "Foo",
	}
	return i
}

func connectToWebSocketServer(wg *sync.WaitGroup) {
	defer wg.Done()
	i := createIdentity()
	fmt.Println("Connecting to WS server with identity:", i.AccountID, i.AccountNumber, i.OrgID, i.UserID)
	res, err := EncodeData(i)
	if err != nil {
		log.Fatal("Unable to generate identity:")
		panic(err)
	}
	// WebSocket server URL
	url := "ws://localhost:8000/wss/chrome-service/v1/ws"

	// Create a new WebSocket connection
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Cookie": []string{fmt.Sprintf("cs_jwt=%s", res.EncodedString)}})
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}

	// Start a goroutine to handle incoming messages from the WebSocket server
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read message from WebSocket server:", err)
				break
			}
			log.Println("Received message from WebSocket server:", string(message))
		}
	}()

	// Wait for a CTRL+C signal to close the connection
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	// Close the WebSocket connection
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("Failed to send close message to WebSocket server:", err)
	}

	// Wait for the WebSocket server to close the connection
	time.Sleep(2 * time.Second)
}

func main() {
	// Number of times to call connectToWebSocketServer
	numCalls := 20000

	var wg sync.WaitGroup
	wg.Add(numCalls)
	for i := 0; i < numCalls; i++ {
		time.Sleep(10 * time.Millisecond)
		go connectToWebSocketServer(&wg)
	}

	wg.Wait()
	fmt.Println("Done")
}
