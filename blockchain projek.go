// Kode perubahan 
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

var upgrader = websocket.Upgrader{}
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan interface{})
var mutex = &sync.Mutex{}

// SmartContract represents the chaincode
type SmartContract struct {
	contractapi.Contract
}

// Student represents a student object
type Student struct {
	Name   string `json:"name"`
	School string `json:"school"`
	Status string `json:"status"`
}

// Transaction represents a transaction log
type Transaction struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	StudentID  string `json:"student_id"`
	Timestamp  string `json:"timestamp"`
}

// RegisterStudent adds a new student to the ledger
func (s *SmartContract) RegisterStudent(ctx contractapi.TransactionContextInterface, id string, name string, school string) error {
	if id == "" || name == "" || school == "" {
		return fmt.Errorf("ID, name, and school cannot be empty")
	}

	existing, err := ctx.GetStub().GetState(id)
	if err != nil {
		return fmt.Errorf("failed to read ledger: %v", err)
	}
	if existing != nil {
		return fmt.Errorf("student with ID %s already exists", id)
	}

	student := Student{
		Name:   name,
		School: school,
		Status: "Registered",
	}

	studentAsBytes, err := json.Marshal(student)
	if err != nil {
		return fmt.Errorf("failed to serialize student: %v", err)
	}

	err = ctx.GetStub().PutState(id, studentAsBytes)
	if err != nil {
		return fmt.Errorf("failed to write student to ledger: %v", err)
	}

	transaction := Transaction{
		ID:        "tx_" + id,
		Action:    "Register",
		StudentID: id,
		Timestamp: ctx.GetStub().GetTxTimestamp().String(),
	}
	logTransaction(transaction)

	broadcast <- transaction

	return nil
}

// QueryStudent retrieves a student by ID
func (s *SmartContract) QueryStudent(ctx contractapi.TransactionContextInterface, id string) (*Student, error) {
	studentAsBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read student data: %v", err)
	}

	if studentAsBytes == nil {
		return nil, fmt.Errorf("student with ID %s not found", id)
	}

	student := new(Student)
	err = json.Unmarshal(studentAsBytes, student)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize student data: %v", err)
	}

	return student, nil
}

// logTransaction logs a transaction
func logTransaction(transaction Transaction) {
	fmt.Printf("Transaction Logged: %+v\n", transaction)
}

// WebSocket handler
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	for {
		message, ok := <-broadcast
		if !ok {
			return
		}
		for client := range clients {
			err := client.WriteJSON(message)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// REST API handlers
func registerStudentHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for handling POST request
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func queryStudentHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for handling GET request
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func verifyStudentsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for query verification
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/api/register-student", registerStudentHandler).Methods("POST")
	router.HandleFunc("/api/query-student/{id}", queryStudentHandler).Methods("GET")
	router.HandleFunc("/api/verify-students", verifyStudentsHandler).Methods("GET")
	router.HandleFunc("/ws", handleConnections)

	go func() {
		log.Fatal(http.ListenAndServe(":8000", router))
	}()

	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
