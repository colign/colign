package wiki

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gobenpark/colign/internal/auth"
)

// ImageHandler serves wiki images over plain HTTP GET.
// Route: GET /api/wiki/images/{id}
func ImageHandler(service *Service, jwtManager *auth.JWTManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Verify JWT from Authorization header or cookie
		token := ""
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if cookie, err := r.Cookie("access_token"); err == nil {
			token = cookie.Value
		}

		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if _, err := jwtManager.ValidateAccessToken(token); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse image ID from path: /api/wiki/images/123
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/wiki/images/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "missing image id", http.StatusBadRequest)
			return
		}

		imageID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "invalid image id", http.StatusBadRequest)
			return
		}

		img, err := service.GetImage(r.Context(), imageID)
		if err != nil {
			if err == ErrImageNotFound {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", img.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(img.Data)))
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(img.Data)
	})
}
