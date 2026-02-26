package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"github.com/gorilla/mux"
)

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

var (
	books     = make(map[string]Book)
	mutex     sync.Mutex
	idCounter = 1
	token     = "secret-token"
)

func main() {
	r := mux.NewRouter()

	// Level 1
	r.HandleFunc("/ping", pingHandler).Methods("GET")

	// Level 2
	r.HandleFunc("/echo", echoHandler).Methods("POST")

	// Level 3 & 4
	// Protected routes
	booksRouter := r.PathPrefix("/books").Subrouter()
	booksRouter.Use(authMiddleware)

	booksRouter.HandleFunc("", getBooks).Methods("GET")
	booksRouter.HandleFunc("", createBook).Methods("POST")
	booksRouter.HandleFunc("/{id}", getBookByID).Methods("GET")
	booksRouter.HandleFunc("/{id}", updateBook).Methods("PUT")
	booksRouter.HandleFunc("/{id}", deleteBook).Methods("DELETE")

	// Level 5 (belum dipakai dulu)
	r.HandleFunc("/auth/token", authHandler).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

/* =========================
   Level 1
========================= */

func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp, _ := json.Marshal(map[string]bool{
		"success": true,
	})
	w.Write(resp)
}

/* =========================
   Level 2
========================= */

func echoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// validate JSON but DO NOT modify
	var js json.RawMessage
	if err := json.Unmarshal(body, &js); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Write(body)
}

/* =========================
   Level 3
========================= */

func createBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	title, ok1 := input["title"].(string)
	author, ok2 := input["author"].(string)
	yearFloat, ok3 := input["year"].(float64)

	if !ok1 || !ok2 || !ok3 {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	id := strconv.Itoa(idCounter)
	idCounter++

	book := Book{
		ID:     id,
		Title:  title,
		Author: author,
		Year:   int(yearFloat),
	}

	books[id] = book

	w.WriteHeader(http.StatusCreated)

	resp, _ := json.Marshal(book)
	w.Write(resp)
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	mutex.Lock()
	defer mutex.Unlock()

	authorFilter := r.URL.Query().Get("author")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Default pagination values
	page := 1
	limit := len(books)

	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	// Step 1: Filtering
	filtered := make([]Book, 0)
	for _, book := range books {
		if authorFilter != "" {
			if book.Author == authorFilter {
				filtered = append(filtered, book)
			}
		} else {
			filtered = append(filtered, book)
		}
	}

	// Step 2: Pagination
	start := (page - 1) * limit
	end := start + limit

	if start > len(filtered) {
		start = len(filtered)
	}

	if end > len(filtered) {
		end = len(filtered)
	}

	result := filtered[start:end]

	resp, _ := json.Marshal(result)
	w.Write(resp)
}

func getBookByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := mux.Vars(r)["id"]

	mutex.Lock()
	defer mutex.Unlock()

	book, exists := books[id]
	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	resp, _ := json.Marshal(book)
	w.Write(resp)
}

/* =========================
   Level 4
========================= */

func updateBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := mux.Vars(r)["id"]

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	title, ok1 := input["title"].(string)
	author, ok2 := input["author"].(string)
	yearFloat, ok3 := input["year"].(float64)

	if !ok1 || !ok2 || !ok3 {
		http.Error(w, "Missing fields", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	// 🔥 If book does not exist, create it
	if _, exists := books[id]; !exists {
		books[id] = Book{
			ID:     id,
			Title:  title,
			Author: author,
			Year:   int(yearFloat),
		}
	} else {
		books[id] = Book{
			ID:     id,
			Title:  title,
			Author: author,
			Year:   int(yearFloat),
		}
	}

	resp, _ := json.Marshal(books[id])
	w.Write(resp)
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := books[id]
	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	delete(books, id)

	// 204 without body
	w.WriteHeader(http.StatusNoContent)
}

/* =========================
   Level 5
========================= */

func authHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var input map[string]string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	username := input["username"]
	password := input["password"]

	if username != "admin" || password != "password" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, _ := json.Marshal(map[string]string{
		"token": token,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expected := "Bearer " + token
		if authHeader != expected {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}