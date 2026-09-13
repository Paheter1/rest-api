package main

import (
	"context"
	"encoding/json"
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
		w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK) /// должно быть ниже w header не нужна отправка кодер сам послает автоматически ok and bad
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
		http.Error(w, "Некоректный JSON", http.StatusBadRequest)
		return
	}

	user.ID = nextID
	nextID++
	users = append(users, user)

	fmt.Println("========================================================")
	fmt.Println("Новый пользователь: " + user.Name + "\nВозраст: " + strconv.Itoa(user.Age) + "\nID: " + strconv.Itoa(user.ID))
	fmt.Println("========================================================")

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
	case "DELETE":

		if !deleteUser(id) {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK) // правильно тут
		fmt.Fprintln(w, "Пользователь удален")
		//w.WriteHeader(http.StatusOK) /// 200 status

		return

	case "PUT":
		updateUserHandler(w, r, id)
		fmt.Println("Пользователь обновлен")

		return

	case "PATCH":
		patchUserHandler(w, r, id)
		return
	}

	for _, user := range users {
		if user.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	http.Error(w, "Пользователь не найден", http.StatusNotFound)
}

func deleteUser(id int) bool {
	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			return true
		}
	}
	return false
}
func updateUserHandler(w http.ResponseWriter, r *http.Request, id int) {
	var user User

	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	for i := range users {
		if users[i].ID == id {
			users[i].Name = user.Name
			users[i].Age = user.Age

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // правильно тут а не в кейсе
			json.NewEncoder(w).Encode(users[i])
			fmt.Println(user)
			return
		}
	}
	http.Error(w, "Пользователь не найден", http.StatusNotFound)
}
func patchUserHandler(w http.ResponseWriter, r *http.Request, id int) {
	var update UpdateUser

	err := json.NewDecoder(r.Body).Decode(&update)

	if err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	for i := range users {
		if users[i].ID == id {
			if update.Name != nil {
				users[i].Name = *update.Name
			}

			if update.Age != nil {
				users[i].Age = *update.Age
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(users[i])
			return
		}
	}

	http.Error(w, "Пользователь не найден", http.StatusNotFound)
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

	fmt.Println("Server start on :8080")
	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		fmt.Println("Server error:", err)
	}

}
