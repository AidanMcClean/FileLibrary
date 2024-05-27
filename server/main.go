package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var db *sql.DB

func main() {
	//var err error
	/*
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	*/
	log.Println("starting application")
	print("reached print in main\n")

	db = initDB()
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	log.Printf("Using port: %s\n", port)

	router := mux.NewRouter()
	router.HandleFunc("/api/categories", getCategories).Methods("GET")
	router.HandleFunc("/api/categories", postCategory).Methods("POST")
	router.HandleFunc("/api/pdfs", getPdfs).Methods("GET")
	router.HandleFunc("/api/pdfs/{id}", downloadPdf).Methods("GET")
	router.HandleFunc("/api/remove-pdfs", removePdfs).Methods("POST")

	router.Handle("/api/pdfs", apiKeyMiddleware(http.HandlerFunc(postPdf))).Methods("POST")

	router.HandleFunc("/api/test", testEndpoint).Methods("GET")

	log.Println("Past the routing")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all origins
		AllowCredentials: true,
		AllowedHeaders:   []string{"Content-Type", "Origin", "Accept", "*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		Debug:            true,
	})

	handler := corsHandler.Handler(router)

	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func testEndpoint(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, world 3!"))
}

func getBlobServiceClient() *azblob.Client {
	connectionString, ok := os.LookupEnv("AZURE_STORAGEBLOB_CONNECTIONSTRING")
	if !ok {
		log.Fatal("the environment variable 'AZURE_STORAGE_CONNECTION_STRING' could not be found")
	}

	serviceClient, err := azblob.NewClientFromConnectionString(connectionString, nil)
	handleError(err)
	return serviceClient
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
	log.Println("Received POST request to add category")

	var category map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		http.Error(w, "Request Error: unable to decode JSON", http.StatusBadRequest)
		return
	}

	name, ok := category["name"].(string)
	if !ok || name == "" {
		log.Println("Invalid or missing category name")
		http.Error(w, "Request Error: invalid category name", http.StatusBadRequest)
		return
	}

	query := "INSERT INTO categories (name) VALUES (@name)"
	_, err := db.Exec(query, sql.Named("name", name))
	if err != nil {
		log.Printf("Error inserting category into database: %v", err)
		http.Error(w, "Server Error: failed to insert category", http.StatusInternalServerError)
		return
	}

	log.Println("Category inserted successfully")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Category created successfully"))
}

// pdf handlers
func getPdfs(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, category_id FROM pdfs")
	if err != nil {
		http.Error(w, "Failed to retrieve records: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pdfs []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		var category_id int
		err := rows.Scan(&id, &name, &category_id)
		if err != nil {
			http.Error(w, "Failed to read record: "+err.Error(), http.StatusInternalServerError)
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

func postPdf(w http.ResponseWriter, r *http.Request) {
	client := getBlobServiceClient()

	err := r.ParseMultipartForm(20 << 20) // limit to 20 MB
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		http.Error(w, "Error parsing multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("pdf")
	if err != nil {
		log.Printf("Error retrieving the file from the form: %v", err)
		http.Error(w, "Invalid file upload", http.StatusBadRequest)
		return
	}
	defer file.Close()

	displayName := r.FormValue("display_name")
	if displayName == "" {
		http.Error(w, "Display name must be provided", http.StatusBadRequest)
		return
	}

	displayName = ensurePDFExtension(displayName)

	categoryID := r.FormValue("category_id")

	query := "INSERT INTO pdfs (name, category_id) OUTPUT INSERTED.id VALUES (@name, @category_id)"
	var newID int
	err = db.QueryRow(query, sql.Named("name", displayName), sql.Named("category_id", categoryID)).Scan(&newID)
	if err != nil {
		log.Printf("Error inserting PDF record into database: %v", err)
		http.Error(w, "Failed to insert PDF record", http.StatusInternalServerError)
		return
	}

	blobName := fmt.Sprintf("%d.pdf", newID)

	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		http.Error(w, "AZURE_STORAGE_CONTAINER_NAME is not set", http.StatusInternalServerError)
		return
	}

	_, err = client.UploadStream(context.TODO(), containerName, blobName, file, nil)
	if err != nil {
		log.Printf("Failed to upload blob: %v", err)
		http.Error(w, "Failed to upload PDF", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("PDF uploaded and recorded successfully with ID: %d", newID)))
}

func ensurePDFExtension(fileName string) string {
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		fileName += ".pdf"
	}
	return fileName
}

func downloadPdf(w http.ResponseWriter, r *http.Request) {
	client := getBlobServiceClient()

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID parameter is required", http.StatusBadRequest)
		return
	}

	var name string
	row := db.QueryRow("SELECT name FROM pdfs WHERE id = @id", sql.Named("id", id))
	if err := row.Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No PDF found with the given ID", http.StatusNotFound)
		} else {
			http.Error(w, "Database query failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		http.Error(w, "AZURE_STORAGE_CONTAINER_NAME is not set", http.StatusInternalServerError)
		return
	}

	blobName := id + ".pdf"

	blobDownloadResponse, err := client.DownloadStream(context.Background(), containerName, blobName, nil)
	if err != nil {
		log.Printf("Failed to download blob: %v", err)
		http.Error(w, "Failed to download PDF", http.StatusInternalServerError)
		return
	}
	defer blobDownloadResponse.Body.Close()

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name+".pdf"))
	if _, err := io.Copy(w, blobDownloadResponse.Body); err != nil {
		log.Printf("Failed to send PDF: %v", err)
		http.Error(w, "Failed to send PDF", http.StatusInternalServerError)
	}
}

func removePdfs(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		PdfIds []int `json:"pdfIds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	client := getBlobServiceClient()
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER_NAME")
	if containerName == "" {
		http.Error(w, "AZURE_STORAGE_CONTAINER_NAME is not set", http.StatusInternalServerError)
		return
	}

	for _, id := range requestData.PdfIds {
		blobName := fmt.Sprintf("%d.pdf", id)

		_, err := client.DeleteBlob(context.Background(), containerName, blobName, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete blob with ID %d: %v", id, err), http.StatusInternalServerError)
			continue
		}

		query := "DELETE FROM pdfs WHERE id = @id"
		_, err = db.Exec(query, sql.Named("id", id))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to delete PDF record with ID %d from database: %v", id, err), http.StatusInternalServerError)
			continue
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PDFs successfully deleted"))
}

// start database
func initDB() *sql.DB {
	connString := os.Getenv("AZURE_SQL_CONNECTIONSTRING")
	if connString == "" {
		log.Fatal("Environment variable AZURE_SQL_CONNECTIONSTRING is not set.")
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		log.Fatalf("Error creating connection pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	log.Println("Connected to Azure SQL Database successfully.")
	return db
}

// check azure API key
func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-api-key")
		expectedApiKey := os.Getenv("API_KEY")
		if apiKey != expectedApiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
