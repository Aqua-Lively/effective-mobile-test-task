package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/joho/godotenv"
)

type Car struct {
	ID     uint   `gorm:"primary_key"`
	RegNum string `json:"regNum"`
	Mark   string `json:"mark"`
	Model  string `json:"model"`
	Year   int    `json:"year"`
	Owner  People `json:"owner"`
}

type People struct {
	ID         uint   `gorm:"primary_key"`
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Patronymic string `json:"patronymic"`
}

var db *gorm.DB

func main() {

	err_start := godotenv.Load(".env")
	if err_start != nil {
		log.Fatal("Error loading .env file")
	}

	driver := os.Getenv("DRIVER")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	dbname := os.Getenv("POSTGRES_DB")
	password := os.Getenv("POSTGRES_PASSWORD")

	connect_string := "host=" + host + " port=" + port + " user=" + user + " dbname=" + dbname + " password=" + password + " sslmode=disable"

	var err error
	db, err = gorm.Open(driver, connect_string)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	db.AutoMigrate(&Car{}, &People{})

	router := mux.NewRouter()

	router.HandleFunc("/cars", getCars).Methods("GET")
	router.HandleFunc("/cars/{id}", deleteCar).Methods("DELETE")
	router.HandleFunc("/cars/{id}", updateCar).Methods("PUT")
	router.HandleFunc("/cars", addCar).Methods("POST")

	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getCars(w http.ResponseWriter, r *http.Request) {
	log.Println("GET request received for getting cars")

	// Извлечение параметров фильтрации из запроса
	queryParams := r.URL.Query()
	var filter Car
	if val, ok := queryParams["regNum"]; ok {
		filter.RegNum = val[0]
	}
	if val, ok := queryParams["mark"]; ok {
		filter.Mark = val[0]
	}
	if val, ok := queryParams["model"]; ok {
		filter.Model = val[0]
	}
	if val, ok := queryParams["year"]; ok {
		year, err := strconv.Atoi(val[0])
		if err == nil {
			filter.Year = year
		}
	}

	// Извлечение параметров пагинации из запроса
	page, err := strconv.Atoi(queryParams.Get("page"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(queryParams.Get("pageSize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// Применение фильтра и пагинации к результатам
	var cars []Car
	result := db.Where(&filter).Limit(pageSize).Offset(offset).Find(&cars)
	if result.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(cars)
}

func deleteCar(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	log.Println("DELETE request received for car with ID:", id)

	var car Car
	db.First(&car, id)

	if car.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	db.Delete(&car)
	w.WriteHeader(http.StatusNoContent)
}

func updateCar(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	log.Println("PUT request received for updating car with ID:", id)

	var car Car
	db.First(&car, id)

	if car.ID == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Обновление полей автомобиля с помощью данных из запроса
	json.NewDecoder(r.Body).Decode(&car)
	db.Save(&car)

	json.NewEncoder(w).Encode(car)
}

func addCar(w http.ResponseWriter, r *http.Request) {
	log.Println("POST request received for adding a new car")

	var car Car
	json.NewDecoder(r.Body).Decode(&car)

	// Проверка наличия владельца в базе данных
	var existingOwner People
	db.Where("name = ? AND surname = ?", car.Owner.Name, car.Owner.Surname).First(&existingOwner)
	if existingOwner.ID == 0 {
		// Если владелец отсутствует, добавляем его
		db.Create(&car.Owner)
	} else {
		car.Owner = existingOwner
	}

	db.Create(&car)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(car)
}
