package main

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

func GetHandler(c *websocket.Conn, p Payload) {
	var data *Payload

	if p.RegExp != "" {
		var results []string
		results, _ = getRecords(p.RegExp)

		b, err := json.Marshal(&results)
		if err != nil {
			data = &Payload{Error: err.Error(), Action: "GET", ID: p.ID}
		} else {
			data = &Payload{Key: p.Key, Action: "GET", Data: string(b), ID: p.ID, TTL: p.TTL}
		}
	} else {
		var result Store
		result, err := getRecord(p.Key)
		if err != nil {
			data = &Payload{Error: err.Error(), Action: "GET", ID: p.ID}
		} else {
			data = &Payload{Key: p.Key, Action: "GET", Data: result.Record, ID: p.ID, TTL: p.TTL}
		}
	}

	websocket.WriteJSON(c, &data)
}

func SetAndScheduleHandler(c *websocket.Conn, p Payload) {
	err := saveRecord(p)
	var data *Payload
	if err != nil {
		data = &Payload{Key: p.Key, Error: err.Error(), Action: "RESPONSE", ID: p.ID}
	} else {
		data = &Payload{Key: p.Key, Action: "RESPONSE", ID: p.ID}
	}
	websocket.WriteJSON(c, &data)
}
func AddToBatch(c *websocket.Conn, p Payload) {
	err := addToBatch(p)
	var data *Payload
	if err != nil {
		data = &Payload{Key: p.Key, Error: err.Error(), Action: "RESPONSE", ID: p.ID}
	} else {
		data = &Payload{Key: p.Key, Action: "RESPONSE", ID: p.ID}
	}
	websocket.WriteJSON(c, &data)
}
func ResetHandler(c *websocket.Conn, p Payload) {

	err := reset()
	var data *Payload
	if err != nil {
		data = &Payload{Key: p.Key, Error: err.Error(), Action: "RESET", ID: p.ID}
	} else {
		data = &Payload{Key: p.Key, Action: "RESET", ID: p.ID}
	}
	websocket.WriteJSON(c, &data)
}

func DeleteHandler(c *websocket.Conn, p Payload) {
	err := deleteRecord(p.Key)
	var data *Payload
	if err != nil {
		data = &Payload{Key: p.Key, Error: err.Error(), Action: "DELETE", ID: p.ID}
	} else {
		data = &Payload{Key: p.Key, Action: "DELETE", ID: p.ID}
	}
	websocket.WriteJSON(c, &data)
}

func FetchKeysHandler(c *websocket.Conn, p Payload) {

	results, err := getKeys()
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
