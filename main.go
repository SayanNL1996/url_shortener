package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ventu-io/go-shortid"
)

const URL string = `^((ftp|http|https):\/\/)?(\S+(:\S*)?@)?((([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(((([a-z\x{00a1}-\x{ffff}0-9]+-?-?_?)*[a-z\x{00a1}-\x{ffff}0-9]+)\.)?)?(([a-z\x{00a1}-\x{ffff}0-9]+-?-?_?)*[a-z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-z\x{00a1}-\x{ffff}]{2,}))?)|localhost)(:(\d{1,5}))?((\/|\?|#)[^\s]*)?$`

var Client *redis.Client

type Url struct {
	LongURL  string `json:"longurl"`
	ShortURL string `json:"shorturl"`
}

type Responsestruct struct {
	Message  string
	Response Url
}

// Redis Connection
func Db() {

	Client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	pong, err := Client.Ping(Client.Context()).Result()
	if err != nil {
		log.Fatal("Could not connect to Redis Server.")
	}
	fmt.Println(pong, "Successfully connected to Redis Server.")

}

// Validating URL
func Matches(str, pattern string) bool {
	match, _ := regexp.MatchString(pattern, str)
	return match
}

func Createurl(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	var url Url
	_ = json.NewDecoder(r.Body).Decode(&url)
	//
	var res = Matches(url.LongURL, URL)
	if res != true {
		fmt.Println("Invalid URL")
		return

	}
	// Generates a hash value for the actual url
	hashcode, err := shortid.Generate()
	if err != nil {
		log.Fatal("Error!!")
	} else {
		// Sets the hash value as key in Redis Server
		Client.Set(Client.Context(), hashcode, url.LongURL, 10*time.Minute).Err()
		var responsestruct = Responsestruct{
			Message: "Short URL generated",
			Response: Url{
				LongURL:  url.LongURL,
				ShortURL: r.Host + "/" + hashcode,
			},
		}
		jsonResponse, err := json.Marshal(responsestruct)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	}
}

// This function will redirect the short url to the actual url
func Redirecturl(w http.ResponseWriter, r *http.Request) {

	shorturl := mux.Vars(r)["shorturl"]
	if shorturl == " " {
		fmt.Println("Error")
	} else {
		// Gets the original url from the short url.
		LongURL, err := Client.Get(Client.Context(), shorturl).Result()
		if err == redis.Nil {
			fmt.Println("no value found")
		} else if err != nil {
			panic(err)
		} else {
			http.Redirect(w, r, LongURL, http.StatusSeeOther)
		}

	}
}

func main() {

	// Init Router
	r := mux.NewRouter()

	// Calling Redis connection function
	Db()

	// Route Handlers / Endpoints
	r.HandleFunc("/api/urlget", Createurl).Methods("POST")
	r.HandleFunc("/{shorturl}", Redirecturl).Methods("GET")

	// Run the server
	log.Println("Server will start at http://localhost:9001/")
	log.Fatal(http.ListenAndServe(":9001", r))

}
