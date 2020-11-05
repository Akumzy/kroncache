package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/timshannon/badgerhold"
)

type Store struct {
	ID     string
	Expire time.Time
	Record string
}
type Payload struct {
	Key    string    `json:"key,omitempty"`
	Expire time.Time `json:"expire,omitempty"`
	Data   string    `json:"data,omitempty"`
	Action string    `json:"action,omitempty"`
}

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options
var DIR, _ = os.Getwd()
var dbDir = filepath.Join(DIR, "db")
var ActiveClient = 1
var idLock = &sync.Mutex{}
var store *badgerhold.Store

const (
	duration = 5 * time.Second
)

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {

		var p Payload
		err := websocket.ReadJSON(c, &p)
		if err != nil {
			log.Println("write:", err)
			break
		}
		fmt.Println(p)
	}
}
func makeID() int {
	idLock.Lock()
	defer idLock.Unlock()
	id := ActiveClient
	ActiveClient++
	return id
}

func main() {

	options := badgerhold.DefaultOptions
	options.Dir = dbDir
	options.ValueDir = dbDir
	var err error
	store, err = badgerhold.Open(options)
	defer store.Close()
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	go runner(store)
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}

}
func saveRecord(p Payload) error {
	err := store.Insert(p.Key, p)
	if err != nil {
		if err == badgerhold.ErrKeyExists {
			err = store.Update(p.Key, p)
		}
	}
	return err
}
func getRecord(key string) (Payload, error) {
	var result Payload
	err := store.Get(key, &result)
	return result, err
}

func runner(store *badgerhold.Store) {
	timer := time.NewTicker(duration)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			{
				fmt.Println(time.Now().Date())
				var data []Store
				err := store.Find(&data, badgerhold.Where("Expire").Le(time.Now()))
				if err != nil {
					log.Fatalln("Find error:", err)
				}
				fmt.Println(data)
			}
		}
	}
}
