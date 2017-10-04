package server

import (
	"reflect"
	"testing"

	"github.com/gothyra/thyra/area"
)

func TestGodPrintRoom(t *testing.T) {
	tests := []struct {
		name string

		clients   []Client
		roomsMap  map[string]map[string][][]area.Cube
		msg       string
		globalMsg string

		expectedScreen Screen
	}{
		// TODO: Add test cases.
		{
			name: "empty room",

			clients:   []Client{},
			roomsMap:  make(map[string]map[string][][]area.Cube),
			msg:       "",
			globalMsg: "",

			expectedScreen: Screen{},
		},
	}

	for _, test := range tests {
		s := Server{Areas: make(map[string]area.Area)}
		got := s.godPrintRoom(test.clients, test.roomsMap, test.msg, test.globalMsg)
		if !reflect.DeepEqual(got, test.expectedScreen) {
			t.Errorf("expected screen: %v\ngot: %v", test.expectedScreen, got)
		}
	}
}
