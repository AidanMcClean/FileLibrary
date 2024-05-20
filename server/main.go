package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

var db *sql.DB

const pdfBasePath = "./PDFlist/"

func main() {
	var err error

	if _, err := os.Stat(pdfBasePath); os.IsNotExist(err) {
		os.Mkdir(pdfBasePath, 0755)
	}

	db, err = initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := mux.NewRouter()
	router.HandleFunc("/api/categories", getCategories).Methods("GET")
	router.HandleFunc("/api/categories", postCategory).Methods("POST")
	router.HandleFunc("/api/pdfs", getPdfs).Methods("GET")
	router.HandleFunc("/api/pdfs", postPdf).Methods("POST")
	router.HandleFunc("/api/pdfs/{id}", downloadPdf).Methods("GET")
	router.HandleFunc("/api/remove-pdfs", removePdfs).Methods("POST")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, //React
		AllowCredentials: true,
		Debug:            true,
	})

	handler := corsHandler.Handler(router)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// category handlers
func getCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		http.Error(w, "Server side error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			http.Error(w, "Server side error", http.StatusInternalServerError)
			return
		}
		categories = append(categories, map[string]interface{}{"id": id, "name": name})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func postCategory(w http.ResponseWriter, r *http.Request) {
	var category map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Request Error", http.StatusBadRequest)
		return
	}

	name, ok := category["name"].(string)
	if !ok || name == "" {
		http.Error(w, "Request Error", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO categories (name) VALUES (?)", name)
	if err != nil {
		http.Error(w, "Server side error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// pdf handlers
func getPdfs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, category_id FROM pdfs")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pdfs []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		var category_id int
		if err := rows.Scan(&id, &name, &category_id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		pdfs = append(pdfs, map[string]interface{}{
			"id":          id,
			"name":        name,
			"category_id": category_id,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pdfs)
}

func cleanFileName(fileName string) string {
	return url.PathEscape(fileName)
}

func postPdf(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("pdf")
	if err != nil {
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	displayName := cleanFileName(r.FormValue("display_name"))
	categoryID := r.FormValue("category_id")

	tempFilePath := filepath.Join(pdfBasePath, "temp_"+fmt.Sprintf("%v_%s", time.Now().UnixNano(), handler.Filename))
	dst, err := os.Create(tempFilePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(dst, file)
	dst.Close()
	if err != nil {
		os.Remove(tempFilePath)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	result, err := db.Exec("INSERT INTO pdfs (name, category_id) VALUES (?, ?)", displayName, categoryID)
	if err != nil {
		os.Remove(tempFilePath)
		http.Error(w, "Failed to save PDF data", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		os.Remove(tempFilePath)
		http.Error(w, "Failed to retrieve last insert ID", http.StatusInternalServerError)
		return
	}

	newPath := filepath.Join(pdfBasePath, fmt.Sprintf("%d%s", id, filepath.Ext(handler.Filename)))
	err = os.Rename(tempFilePath, newPath)
	if err != nil {
		http.Error(w, "Failed to rename file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("File uploaded successfully"))
}

func downloadPdf(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var fileName string
	err := db.QueryRow("SELECT name FROM pdfs WHERE id = ?", id).Scan(&fileName)
	if err != nil {
		http.Error(w, "PDF not found in database", http.StatusNotFound)
		return
	}

	filePath := filepath.Join(pdfBasePath, fmt.Sprintf("%s.pdf", id))

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found on server", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, file)
}

func removePdfs(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		PdfIds []int `json:"pdfIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	for _, id := range requestData.PdfIds {
		filePath := filepath.Join(pdfBasePath, fmt.Sprintf("%d.pdf", id))

		if err := os.Remove(filePath); err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete file: %s", filePath), http.StatusInternalServerError)
			continue
		}

		_, err := db.Exec("DELETE FROM pdfs WHERE id = ?", id)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete PDF record with ID %d from database", id), http.StatusInternalServerError)
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
}

// start database
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./library.db")
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	initializeTables(db)

	return db, nil
}

// initialize database
func initializeTables(db *sql.DB) {
	createCategoriesTable := `
    CREATE TABLE IF NOT EXISTS categories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT UNIQUE NOT NULL
    );`
	_, err := db.Exec(createCategoriesTable)
	if err != nil {
		log.Fatalf("Failed to create categories table: %v", err)
	}

	createPDFsTable := `
    CREATE TABLE IF NOT EXISTS pdfs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        category_id INTEGER,
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET DEFAULT
    );`
	_, err = db.Exec(createPDFsTable)
	if err != nil {
		log.Fatalf("Failed to create pdfs table: %v", err)
	}
}
