package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Handler untuk upload file
func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // Batas ukuran file: 10 MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Ambil file dari form
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Buat folder upload jika belum ada
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Simpan file ke sistem
	filePath := filepath.Join(uploadDir, handler.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	// Simpan hash file ke blockchain (contoh sederhana)
	fileHash := "QmcNEnutajYqdYBVrMbW7grHiranYACfdUZk75d2ccR6yW" // Ganti dengan hash sebenarnya (misalnya, dari IPFS)
	ctx := &contractapi.TransactionContext{}
	contract := SmartContract{}
	if err := contract.RegisterFile(ctx, "file-id", handler.Filename, fileHash); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Beri respons ke client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origin (untuk development saja)
	},
}
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
	ID        string `json:"id"`
	Action    string `json:"action"`
	StudentID string `json:"student_id"`
	Timestamp string `json:"timestamp"`
}

// RegisterFile adds a file hash to the ledger
func (s *SmartContract) RegisterFile(ctx contractapi.TransactionContextInterface, id string, filename string, hash string) error {
	if id == "" || filename == "" || hash == "" {
		return fmt.Errorf("ID, filename, and hash cannot be empty")
	}

	file := struct {
		Filename string `json:"filename"`
		Hash     string `json:"hash"`
	}{
		Filename: filename,
		Hash:     hash,
	}

	fileAsBytes, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to serialize file: %v", err)
	}

	err = ctx.GetStub().PutState(id, fileAsBytes)
	if err != nil {
		return fmt.Errorf("failed to write file to ledger: %v", err)
	}

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
	if err := contract.RegisterFile(ctx, req.ID, req.Name, req.School); err != nil {
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

	//route untuk upload files
	r.Post("/upload", uploadFileHandler)

	// Serve static files
	staticDir := http.Dir(filepath.Join("static"))
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticDir)))

	// Root route untuk menampilkan index.html
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("static", "index.html"))
	})

	// API routes
	r.Post("/api/register-student", registerStudentHandler)
	r.Get("/api/query-student/{id}", queryStudentHandler)
	r.Get("/api/verify-students", verifyStudentsHandler)
	r.Get("/ws", handleConnections)

	// Jalankan server
	go func() {
		log.Fatal(http.ListenAndServe(":8000", r))
	}()

	// Broadcast transaksi ke WebSocket clients
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
