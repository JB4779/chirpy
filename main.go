package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/JB4779/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.dbQueries.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't delete users",
		)
		return
	}

	cfg.fileserverHits.Store(0)

	w.WriteHeader(http.StatusOK)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	respBody := errorResponse{
		Error: msg,
	}

	respondWithJSON(w, code, respBody)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	// Decode the incoming JSON request
	decoder := json.NewDecoder(r.Body)

	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Couldn't decode parameters",
		)
		return
	}

	// Validate chirp length
	if len(params.Body) > 140 {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Chirp is too long",
		)
		return
	}

	// Clean prohibited words
	cleanedBody := cleanBody(params.Body)

	// Save the chirp to PostgreSQL using SQLC
	dbChirp, err := cfg.dbQueries.CreateChirp(
		r.Context(),
		database.CreateChirpParams{
			Body:   cleanedBody,
			UserID: params.UserID,
		},
	)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't create chirp",
		)
		return
	}

	// Convert the database Chirp into our API Chirp
	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	// Return the newly created chirp
	respondWithJSON(
		w,
		http.StatusCreated,
		chirp,
	)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)

	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Couldn't decode parameters",
		)
		return
	}

	dbUser, err := cfg.dbQueries.CreateUser(
		r.Context(),
		params.Email,
	)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't create user",
		)
		return
	}

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	respondWithJSON(
		w,
		http.StatusCreated,
		user,
	)
}

func cleanBody(body string) string {
	words := strings.Split(body, " ")

	for i, word := range words {
		switch strings.ToLower(word) {
		case "kerfuffle", "sharbert", "fornax":
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't get chirps",
		)
		return
	}

	chirps := []Chirp{}

	for _, dbChirp := range dbChirps {
		chirp := Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		}

		chirps = append(chirps, chirp)
	}

	respondWithJSON(
		w,
		http.StatusOK,
		chirps,
	)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")

	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		respondWithError(
			w,
			http.StatusNotFound,
			"Chirp not found",
		)
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirp(
		r.Context(),
		chirpID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(
			w,
			http.StatusNotFound,
			"Chirp not found",
		)
		return
	}
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't get chirp",
		)
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(
		w,
		http.StatusOK,
		chirp,
	)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	const port = ":8080"

	apiCfg := &apiConfig{
		dbQueries: dbQueries,
		platform:  platform,
	}

	// create ServeMux
	mux := http.NewServeMux()

	handler := http.StripPrefix(
		"/app",
		http.FileServer(http.Dir(".")),
	)

	// register handlers
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("GET  /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("GET  /api/healthz", handlerReadiness)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)

	// create http.Server using the mux and port 8080
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// start the server
	//server.ListenAndServe()
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
