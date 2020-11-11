package main

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/timshannon/badgerhold"
)

func TestMain(m *testing.M) {
	go func() {
		os.Exit(m.Run())
	}()
}
func Test_makeID(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "Generate ID for each connection", want: 1},
		{name: "Generate ID for each connection", want: 2},
		{name: "Generate ID for each connection", want: 3},
		{name: "Generate ID for each connection", want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeID(); got != tt.want {
				t.Errorf("makeID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_saveRecord(t *testing.T) {
	type args struct {
		p Payload
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Save a record without key", args: args{p: Payload{}}, wantErr: true},
		{name: "Save a record with key", args: args{p: Payload{Key: "Boo"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := saveRecord(tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("saveRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getRecord(t *testing.T) {
	saveRecord(Payload{Key: "COOL", Expire: time.Now(), Data: "LOKI"})
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		args    args
		want    Payload
		wantErr bool
	}{
		{name: "getRecord", args: args{key: "COOL"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRecord(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getAllRecord(t *testing.T) {
	tests := []struct {
		name    string
		want    []Payload
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAllRecord()
			if (err != nil) != tt.wantErr {
				t.Errorf("getAllRecord() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAllRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_purgeRecords(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := purgeRecords(); (err != nil) != tt.wantErr {
				t.Errorf("purgeRecords() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_runner(t *testing.T) {
	type args struct {
		store *badgerhold.Store
		eb    *EventBus
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner(tt.args.store, tt.args.eb)
		})
	}
}
