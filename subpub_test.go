package main

import (
	"sync"
	"testing"
)

func TestEventBus_Publish(t *testing.T) {
	type fields struct {
		subscribers Subscribers
		rm          sync.RWMutex
	}
	type args struct {
		data Payload
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &EventBus{
				subscribers: tt.fields.subscribers,
				rm:          tt.fields.rm,
			}
			eb.Publish(tt.args.data)
		})
	}
}

func TestEventBus_Unsbscribe(t *testing.T) {
	type fields struct {
		subscribers Subscribers
		rm          sync.RWMutex
	}
	type args struct {
		key int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &EventBus{
				subscribers: tt.fields.subscribers,
				rm:          tt.fields.rm,
			}
			eb.Unsbscribe(tt.args.key)
		})
	}
}
