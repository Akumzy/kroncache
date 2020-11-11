package main

import (
	"encoding/json"
	"errors"
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
	ID      string    `badgerhold:"index" json:"id,omitempty"`
	Expire  time.Time `badgerhold:"index" json:"expire,omitempty"`
	Record  string    `json:"record,omitempty"`
	Expired bool      `json:"expired,omitempty"`
}
type Payload struct {
	Key    string    `json:"key,omitempty"`
	Expire time.Time `json:"expire,omitempty"`
	Data   string    `json:"data,omitempty"`
	Action string    `json:"action,omitempty"`
	Error  string    `json:"error,omitempty"`
	ID     string    `json:"id,omitempty"`
}

var (
	upgrader     = websocket.Upgrader{}
	DIR, _       = os.Getwd()
	dbDir        = filepath.Join(DIR, "db")
	ActiveClient = 1
	idLock       = &sync.Mutex{}
	store        *badgerhold.Store
	events       = make(chan Store)
	eb           = &EventBus{
		subscribers: Subscribers{},
	}
)

const (
	duration = 20 * time.Second
)

func init() {
	startDB()
}
func startDB() {

	options := badgerhold.DefaultOptions
	options.Dir = dbDir
	options.ValueDir = dbDir
	var err error
	store, err = badgerhold.Open(options)

	if err != nil {
		log.Fatal(err)
	}
}
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "5093"
	}
	var addr = flag.String("addr", "localhost:"+port, "http service address")

	go runner(store, eb)

	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}

}

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	key := makeID()
	defer func() {
		c.Close()
		eb.Unsbscribe(key)
	}()

	var ch = make(DataChannel)
	go eb.Subscribe(key, ch)
	go func(c *websocket.Conn, ch DataChannel) {
		for {
			select {
			case event := <-ch:
				websocket.WriteJSON(c, event)
			}
		}

	}(c, ch)

	for {
		var p Payload
		err := websocket.ReadJSON(c, &p)
		if err != nil {
			log.Println("write:", err)
			break
		}
		switch p.Action {
		case "SET":
			{

				err = saveRecord(p)
				var data *Payload
				if err != nil {
					data = &Payload{Key: p.Key, Error: err.Error(), Action: "RESPONSE", ID: p.ID}
				} else {
					data = &Payload{Key: p.Key, Action: "RESPONSE", ID: p.ID}
				}

				err = websocket.WriteJSON(c, &data)
				log.Fatalln(err)
			}
		case "GET":
			{
				var result Store
				result, err = getRecord(p.Key)
				var data *Payload
				if err != nil {
					data = &Payload{Key: p.Key, Error: err.Error(), Action: "GET", ID: p.ID, Expire: p.Expire}
				} else {
					data = &Payload{Key: p.Key, Action: "GET", Data: result.Record, ID: p.ID, Expire: p.Expire}
				}
				websocket.WriteJSON(c, &data)
			}
		case "DELETE":
			{

				err = deleteRecord(p.Key)
				var data *Payload
				if err != nil {
					data = &Payload{Key: p.Key, Error: err.Error(), Action: "DELETE", ID: p.ID}
				} else {
					data = &Payload{Key: p.Key, Action: "DELETE", ID: p.ID}
				}
				websocket.WriteJSON(c, &data)
			}
		case "PURGE":
			{

				err = purgeRecords()
				var data *Payload
				if err != nil {
					data = &Payload{Key: p.Key, Error: err.Error(), Action: "PURGE", ID: p.ID}
				} else {
					data = &Payload{Key: p.Key, Action: "PURGE", ID: p.ID}
				}
				websocket.WriteJSON(c, &data)
			}
		case "DUMP":
			{
				var result []Store
				result, err = getAllRecord()

				var data *Payload
				if err != nil {
					data = &Payload{Error: err.Error(), Action: "DUMP", ID: p.ID}
				} else {
					d, _ := json.Marshal(result)
					data = &Payload{Action: "DUMP", Data: string(d), ID: p.ID}
				}
				websocket.WriteJSON(c, &data)
			}
		}

	}
}
func makeID() int {
	idLock.Lock()
	defer idLock.Unlock()
	id := ActiveClient
	ActiveClient++
	return id
}

func saveRecord(p Payload) error {
	if p.Key == "" {
		return errors.New("Record key can not be empty.")
	}
	record := Store{ID: p.Key, Expire: p.Expire, Record: p.Data}
	err := store.Insert(p.Key, record)
	if err != nil {
		if err == badgerhold.ErrKeyExists {
			err = store.Update(p.Key, record)
		}
	}
	return err
}
func getRecord(key string) (Store, error) {
	var result Store
	if key == "" {
		return result, errors.New("key can not be empty.")
	}

	err := store.Get(key, &result)
	return result, err
}
func getAllRecord() ([]Store, error) {
	var result []Store
	err := store.Find(&result, nil)
	return result, err
}
func purgeRecords() error {
	return store.DeleteMatching(&Store{}, nil)
}

func deleteRecord(key string) error {
	return store.Delete(key, &Store{})
}

func runner(store *badgerhold.Store, eb *EventBus) {
	timer := time.NewTicker(duration)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			{
				var data []Store
				now := time.Now()
				err := store.Find(&data, badgerhold.Where("Expired").Ne(true).And("Expire").Le(now))
				if err != nil {
					log.Fatalln("Find error:", err)
				}
				err = store.UpdateMatching(&Store{}, badgerhold.Where("Expired").Ne(true).And("Expire").Le(now), func(record interface{}) error {
					update, ok := record.(*Store)
					if !ok {
						return fmt.Errorf("Record isn't the correct type!  Wanted Store, got %T", record)
					}
					update.Expired = true
					return nil
				})
				if err != nil {
					log.Fatalln("Find error:", err)
				}
				for _, event := range data {
					eb.Publish(Payload{Key: event.ID, Data: event.Record, Expire: event.Expire, Action: "EXPIRED"})
				}

			}
		}
	}
}
