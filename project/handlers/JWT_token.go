package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var password string

func InitAuth(pass string) {
	password = pass
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем пароль из переменной окружения
		if len(password) > 0 {
			var jwtToken string

			// Получаем токен из куки
			cookie, err := r.Cookie("token")
			if err == nil {
				jwtToken = cookie.Value
			}

			// Проверяем токен
			valid := validateJWT(jwtToken, password)
			if !valid {
				// Возвращаем ошибку 401, если токен невалиден
				http.Error(w, "требуется аутентификация", http.StatusUnauthorized)
				return
			}
		}

		// Если всё в порядке, передаем управление следующему обработчику
		next(w, r)
	})
}

// validateJWT проверяет JWT-токен
func validateJWT(tokenString, password string) bool {
	if tokenString == "" {
		return false
	}

	// Парсим токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем метод подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(password), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	// Проверяем, что хэш пароля в токене совпадает с текущим паролем
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["hash"] != password {
		return false
	}

	return true
}

type SignInRequest struct {
	Password string `json:"password"`
}

type SignInResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func SignInHandler(w http.ResponseWriter, r *http.Request) {
	// Парсим JSON из тела запроса
	var req SignInRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SignInResponse{Error: "Invalid JSON"})
		return
	}

	// Получаем пароль из переменной окружения
	password := os.Getenv("TODO_PASSWORD")
	if password == "" {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SignInResponse{Error: "Аутентификация не настроена"})
		return
	}

	// Проверяем пароль
	if req.Password != password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(SignInResponse{Error: "Неверный пароль"})
		return
	}

	// Создаем JWT-токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"hash": password, // Хэш пароля в полезной нагрузке
		"exp":  time.Now().Add(8 * time.Hour).Unix(),
	})

	// Подписываем токен
	tokenString, err := token.SignedString([]byte(password))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SignInResponse{Error: "Ошибка генерации токена"})
		return
	}

	// Возвращаем токен
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SignInResponse{Token: tokenString})
}
