package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

var (
	books = make(map[string]Book)
	mutex sync.Mutex
	token = "secret-token"
)

func main() {
	r := mux.NewRouter()

	// Level 1
	r.HandleFunc("/ping", pingHandler).Methods("GET")

	// Level 2
	r.HandleFunc("/echo", echoHandler).Methods("POST")

	// Level 3 & 4
	r.HandleFunc("/books", createBook).Methods("POST")
	r.HandleFunc("/books", getBooks).Methods("GET")
	r.HandleFunc("/books/{id}", getBookByID).Methods("GET")
	r.HandleFunc("/books/{id}", updateBook).Methods("PUT")
	r.HandleFunc("/books/{id}", deleteBook).Methods("DELETE")

	// Level 5
	r.HandleFunc("/auth/token", authHandler).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	json.NewEncoder(w).Encode(body)
}

func createBook(w http.ResponseWriter, r *http.Request) {
	var book Book
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if book.Title == "" || book.Author == "" {
		http.Error(w, "Title and Author required", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	book.ID = strconv.Itoa(len(books) + 1)
	books[book.ID] = book

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(book)
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	// Auth check
	if r.Header.Get("Authorization") != "Bearer "+token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	var result []Book

	authorFilter := r.URL.Query().Get("author")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	for _, book := range books {
		if authorFilter == "" || strings.EqualFold(book.Author, authorFilter) {
			result = append(result, book)
		}
	}

	// Pagination
	if pageStr != "" && limitStr != "" {
		page, _ := strconv.Atoi(pageStr)
		limit, _ := strconv.Atoi(limitStr)

		start := (page - 1) * limit
		end := start + limit

		if start > len(result) {
			result = []Book{}
		} else {
			if end > len(result) {
				end = len(result)
			}
			result = result[start:end]
		}
	}

	json.NewEncoder(w).Encode(result)
}

func getBookByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	mutex.Lock()
	defer mutex.Unlock()

	book, exists := books[id]
	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(book)
}

func updateBook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	mutex.Lock()
	defer mutex.Unlock()

	_, exists := books[id]
	if !exists {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var updated Book
	json.NewDecoder(r.Body).Decode(&updated)

	if updated.Title == "" || updated.Author == "" {
		http.Error(w, "Invalid Data", http.StatusBadRequest)
		return
	}

	updated.ID = id
	books[id] = updated

	json.NewEncoder(w).Encode(updated)
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
	w.WriteHeader(http.StatusNoContent)
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}