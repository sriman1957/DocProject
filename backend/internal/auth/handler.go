package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type loginRequest struct {
	CollegeCode string `json:"college_code"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewHandler creates the HTTP handler for authentication.
func NewHandler(
	service *Service,
	tokens *TokenService,
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/login", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		var request loginRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Reject requests containing multiple JSON values.
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		user, err := service.Login(
			r.Context(),
			request.CollegeCode,
			request.Email,
			request.Password,
		)

		if errors.Is(err, ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		if err != nil {
			if logger != nil {
				logger.Error("login failed", "error", err)
			}

			writeError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
			return
		}

		accessToken, err := tokens.Generate(user)
		if err != nil {
			if logger != nil {
				logger.Error("access token generation failed", "error", err)
			}

			writeError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
			return
		}

		writeAuthJSON(w, http.StatusOK, loginResponse{
			AccessToken: accessToken,
			User:        user,
		})
	})

	return mux
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeAuthJSON(w, status, errorResponse{
		Error: message,
	})
}

func writeAuthJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// The response might already be committed, so there is
		// no reliable way to send another HTTP error response.
		return
	}
}
