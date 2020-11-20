package main

import (
	"encoding/json"
	"errors"
	"flag"
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
	TTL     time.Time `badgerhold:"index" json:"ttl,omitempty"`
	Record  string    `json:"record,omitempty"`
	Expired bool      `json:"expired,omitempty"`
	ACK     bool      `json:"ack,omitempty"`
}
type Payload struct {
	Key    string    `json:"key,omitempty"`
	TTL    time.Time `json:"ttl,omitempty"`
	Data   string    `json:"data,omitempty"`
	Action string    `json:"action,omitempty"`
	Error  string    `json:"error,omitempty"`
	ID     string    `json:"id,omitempty"`
	ACK    bool      `json:"ack,omitempty"`
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
	duration = 1 * time.Second
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
				websocket.WriteJSON(c, &data)

			}
		case "GET":
			{
				var result Store
				result, err = getRecord(p.Key)
				var data *Payload
				if err != nil {
					data = &Payload{Error: err.Error(), Action: "GET", ID: p.ID}
				} else {
					data = &Payload{Key: p.Key, Action: "GET", Data: result.Record, ID: p.ID, TTL: p.TTL}
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
		case "RESET":
			{

				err = reset()
				var data *Payload
				if err != nil {
					data = &Payload{Key: p.Key, Error: err.Error(), Action: "RESET", ID: p.ID}
				} else {
					data = &Payload{Key: p.Key, Action: "RESET", ID: p.ID}
				}
				websocket.WriteJSON(c, &data)
			}
		case "KEYS":
			{
				var results []string
				results, err = getKeys()
				var data *Payload
				if err != nil {
					data = &Payload{Error: err.Error(), Action: "KEYS", ID: p.ID}
				} else {
					var b []byte
					b, err = json.Marshal(&results)
					data = &Payload{Action: "KEYS", Data: string(b), ID: p.ID}
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

	record := Store{ID: p.Key, TTL: p.TTL, Record: p.Data, ACK: p.ACK}
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
func getKeys() ([]string, error) {
	var result []Store
	err := store.Find(&result, nil)
	var keys []string
	if err == nil {
		keys := make([]string, len(result))
		for _, r := range result {
			keys = append(keys, r.ID)
		}
		result = nil
		return keys, nil
	}
	return keys, err
}
func reset() error {
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
				err := store.Find(&data, badgerhold.Where("Expired").Ne(true).And("TTL").Le(now))
				if err != nil {
					log.Fatalln("Find error:", err)
				}

				for _, r := range data {
					if r.ACK {
						store.Update(r.ID, &Store{ID: r.ID, Record: r.Record, ACK: r.ACK, TTL: r.TTL, Expired: true})
						eb.Publish(Payload{Key: r.ID, Data: r.Record, TTL: r.TTL, Action: "EXPIRED"})
					} else {
						store.Delete(r.ID, &Store{})
					}

				}

			}
		}
	}
}
