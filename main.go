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
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	cron "github.com/robfig/cron/v3"
	"github.com/timshannon/badgerhold"
)

type Store struct {
	ID      string    `badgerhold:"index" json:"id,omitempty"`
	TTL     time.Time `badgerhold:"index" json:"ttl,omitempty"`
	Record  string    `json:"record,omitempty"`
	Expired bool      `json:"expired,omitempty"`
	ACK     bool      `json:"ack,omitempty"`
	Cron    string    `json:"cron,omitempty"`
	Action  string    `json:"action,omitempty"`
}
type BatchStore struct {
	ID         uint64 `badgerhold:"index" json:"id,omitempty"`
	Record     string `json:"record,omitempty"`
	BatchID    string `json:"batchId,omitempty"`
	BatchGroup time.Time
	Flagged    bool
}
type Payload struct {
	Key    string    `json:"key,omitempty"`
	TTL    time.Time `json:"ttl,omitempty"`
	Data   string    `json:"data,omitempty"`
	Action string    `json:"action,omitempty"`
	Error  string    `json:"error,omitempty"`
	ID     string    `json:"id,omitempty"`
	ACK    bool      `json:"ack,omitempty"`
	Cron   string    `json:"cron,omitempty"`
	RegExp string    `json:"regex,omitempty"`
}
type BatchRecord struct {
	BatchID    string    `json:"batchId,omitempty"`
	BatchGroup time.Time `json:"batchGroup"`
	records    []string
}

var (
	upgrader     = websocket.Upgrader{}
	DIR, _       = os.Getwd()
	dbDir        = filepath.Join(DIR, "db")
	ActiveClient = 1
	idLock       = &sync.Mutex{}
	batchLock    = &sync.Mutex{}
	store        *badgerhold.Store
	events       = make(chan Store)
	eb           = &EventBus{
		subscribers: Subscribers{},
	}
	specParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	// no key error
	ErrNoKey = errors.New("No key specified")
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
	var addr = flag.String("addr", ":"+port, "http service address")

	go runner(store, eb)

	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}

}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("New Connection")
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
		case "SET", "SCHEDULE", "BATCH":
			SetAndScheduleHandler(c, p)
		case "GET":
			GetHandler(c, p)
		case "DELETE":
			DeleteHandler(c, p)
		case "RESET":
			ResetHandler(c, p)
		case "ADD-BATCH":
			AddToBatch(c, p)
		case "KEYS":
			FetchKeysHandler(c, p)
		case "INCREMENT":
			IncrementHandler(c, p)
		case "DECREMENT":
			DecrementHandler(c, p)
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
	record := Store{ID: p.Key, TTL: p.TTL, Record: p.Data, ACK: p.ACK, Cron: p.Cron, Action: p.Action}

	if p.Cron != "" {
		sch, err := specParser.Parse(record.Cron)
		if err != nil {
			return err
		}
		record.TTL = sch.Next(time.Now())
	}
	err := store.Insert(p.Key, record)
	if err != nil {
		if err == badgerhold.ErrKeyExists {
			err = store.Update(p.Key, record)
		}
	}
	return err
}
func addToBatch(p Payload) error {
	batchLock.Lock()
	defer batchLock.Unlock()
	if p.Key == "" {
		return errors.New("Record key can not be empty.")
	}
	record := BatchStore{BatchID: p.Key, Record: p.Data}

	return store.Insert(badgerhold.NextSequence(), record)
}
func getRecords(regex string) ([]string, error) {
	var result []Store

	reg := regexp.MustCompile(regex)
	err := store.Find(&result, badgerhold.Where("ID").RegExp(reg))
	var results []string
	if err == nil {
		results := make([]string, len(results))
		for _, r := range result {
			results = append(results, r.Record)
		}
		return results, nil
	}
	return results, err

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

// increment value by value
func increment(key string, value int) (int, error) {
	var result Store
	err := store.Get(key, &result)
	newValue := value
	if err != nil {
		return 0, err
	}
	oldValue, err := strconv.Atoi(result.Record)
	if err != nil {
		return 0, err
	}
	newValue = oldValue + value

	result.Record = strconv.Itoa(newValue)
	err = store.Update(key, result)

	return newValue, err
}

// decrement value by value
func decrement(key string, value int) (int, error) {
	var result Store
	err := store.Get(key, &result)
	newValue := value
	if err != nil {
		return 0, err
	}
	oldValue, err := strconv.Atoi(result.Record)
	if err != nil {
		return 0, err
	}
	newValue = oldValue - value

	result.Record = strconv.Itoa(newValue)
	err = store.Update(key, result)

	return newValue, err
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
					if r.Cron != "" {
						sch, err := specParser.Parse(r.Cron)
						if err != nil {
							store.Delete(r.ID, &Store{})
						} else {
							nextTime := sch.Next(time.Now())
							store.Update(r.ID, &Store{ID: r.ID, Record: r.Record, TTL: nextTime, Cron: r.Cron, Action: r.Action})
						}
						if r.Action == "BATCH" {
							var records []BatchStore
							updateBatchStore(now, r.ID)

							store.Find(&records, badgerhold.Where("BatchID").Eq(r.ID).And("BatchGroup").Eq(now))

							values := make([]string, len(records))

							for _, r := range records {
								values = append(values, r.Record)
							}
							byts, _ := json.Marshal(values)

							eb.Publish(Payload{Key: r.ID, Data: string(byts), TTL: r.TTL, Action: "BATCH"})
						} else {
							eb.Publish(Payload{Key: r.ID, Data: r.Record, TTL: r.TTL, Action: "CRON"})
						}

					} else if r.ACK {
						store.Update(r.ID, &Store{ID: r.ID, Record: r.Record, ACK: r.ACK, TTL: r.TTL, Expired: true})
						action := "EXPIRED"
						if r.Action == "SCHEDULE" {
							action = "SCHEDULE"
							store.Delete(r.ID, &Store{})
						}
						eb.Publish(Payload{Key: r.ID, Data: r.Record, TTL: r.TTL, Action: action})
					} else {
						store.Delete(r.ID, &Store{})
					}

				}

			}
		}
	}
}

func updateBatchStore(now time.Time, ID string) error {
	batchLock.Lock()
	defer batchLock.Unlock()
	return store.UpdateMatching(&BatchStore{}, badgerhold.Where("BatchID").Eq(ID).And("Flagged").Ne(true), func(record interface{}) error {
		update, ok := record.(*BatchStore)
		if !ok {
			return fmt.Errorf("Record isn't the correct type!  Wanted BatchStore, got %T", record)
		}
		update.Flagged = true
		update.BatchGroup = now

		return nil
	})
}
