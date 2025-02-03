package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Izinkan semua origin (untuk development saja)
	},
}
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan interface{})
var mutex = &sync.Mutex{};

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
	ID        string `json:"id"`
	Action    string `json:"action"`
	StudentID string `json:"student_id"`
	Timestamp string `json:"timestamp"`
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
		Timestamp: time.Now().Format(time.RFC3339),
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
	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		School string `json:"school"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simulate a transaction context (replace with actual Fabric context)
	ctx := &contractapi.TransactionContext{}
	contract := SmartContract{}
	if err := contract.RegisterStudent(ctx, req.ID, req.Name, req.School); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Student registered successfully"))
}

func queryStudentHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	// Simulate a transaction context (replace with actual Fabric context)
	ctx := &contractapi.TransactionContext{}
	contract := SmartContract{}
	student, err := contract.QueryStudent(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

func verifyStudentsHandler(w http.ResponseWriter, r *http.Request) {
	// Placeholder for query verification
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Not implemented"))
}

func main() {
	r := chi.NewRouter()

	r.Post("/api/register-student", registerStudentHandler)
	r.Get("/api/query-student/{id}", queryStudentHandler)
	r.Get("/api/verify-students", verifyStudentsHandler)
	r.Get("/ws", handleConnections)

	go func() {
		log.Fatal(http.ListenAndServe(":8000", r))
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
