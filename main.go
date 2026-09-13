package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type UpdateUser struct {
	Name *string `json:"name"`
	Age  *int    `json:"age"`
}

var db *pgx.Conn

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "HELLO GO")
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	user := User{
		Name: "Pavel",
		Age:  18,
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(user)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		rows, err := db.Query(context.Background(), "SELECT id, name, age FROM users")

		if err != nil {
			http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		defer rows.Close()

		users := []User{}

		for rows.Next() {
			var user User

			err := rows.Scan(&user.ID, &user.Name, &user.Age)

			if err != nil {
				http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
				return
			}
			users = append(users, user)

		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)

	case "POST":
		createUserHandler(w, r)

	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}

}
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var user User

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	err = db.QueryRow(
		context.Background(),
		"INSERT INTO users (name, age) VALUES ($1, $2) RETURNING id",
		user.Name,
		user.Age,
	).Scan(&user.ID)

	if err != nil {
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	fmt.Println("Новый пользователь:", user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(user)
}
func userByIDHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")

	id, err := strconv.Atoi(path)

	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	switch r.Method {

	case "GET":
		var user User

		err = db.QueryRow(
			context.Background(),
			"SELECT id, name, age FROM users WHERE id = $1",
			id,
		).Scan(&user.ID, &user.Name, &user.Age)

		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return

	case "DELETE":
		deleted, err := deleteUser(id)

		if err != nil {
			http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		if !deleted {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Пользователь удален")
		return

	case "PUT":
		updateUserHandler(w, r, id)
		return

	case "PATCH":
		patchUserHandler(w, r, id)
		return

	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
}

func deleteUser(id int) (bool, error) {
	result, err := db.Exec(
		context.Background(),
		"DELETE FROM users WHERE id = $1",
		id,
	)

	if err != nil {
		return false, err
	}

	return result.RowsAffected() > 0, nil
}

func updateUserHandler(w http.ResponseWriter, r *http.Request, id int) {
	var user User

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	err = db.QueryRow(
		context.Background(),
		"UPDATE users SET name = $1, age = $2 WHERE id = $3 RETURNING id, name, age",
		user.Name,
		user.Age,
		id,
	).Scan(&user.ID, &user.Name, &user.Age)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(user)
}
func patchUserHandler(w http.ResponseWriter, r *http.Request, id int) {
	var update UpdateUser

	err := json.NewDecoder(r.Body).Decode(&update)

	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	var user User

	err = db.QueryRow(
		context.Background(),
		`UPDATE users
		 SET name = COALESCE($1, name),
		     age = COALESCE($2, age)
		 WHERE id = $3
		 RETURNING id, name, age`,
		update.Name,
		update.Age,
		id,
	).Scan(&user.ID, &user.Name, &user.Age)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}

		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(user)
}

func main() {
	var err error

	db, err = pgx.Connect(context.Background(), "postgres://rest_api_user:123456@localhost:5432/rest_api")

	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return
	}

	defer db.Close(context.Background())

	http.HandleFunc("/user", userHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/users/", userByIDHandler)

	fmt.Println("Server start on :8080")
	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Println("Server error:", err)
	}

}
