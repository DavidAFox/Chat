package http

import (
	"bytes"
	"github.com/davidafox/chat/client"
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/clientdata/filedata"
	"github.com/davidafox/chat/room"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTPHandlesCORSOptionsRequest(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", "www.example.com", nil)
	if err != nil {
		t.Error("Error creating request in TestServeHTTPHandlesCORSOptionsRequest: ", err)
	}
	w := httptest.NewRecorder()
	wsh := newTestRoomHandler(t)
	wsh.ServeHTTP(w, req)
	expectedOptionsHeaders := []TestHeader{
		{"Access-Control-Allow-Origin", wsh.origin},
		{"Access-Control-Allow-Methods", "POST"},
		{"Access-Control-Allow-Methods", "OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type"},
		{"Access-Control-Max-Age", "1728000"},
		{"Content-Type", "application/json"},
	}
	checkHeadersPresent(w.Header(), expectedOptionsHeaders, t)
}

func checkHeadersPresent(header http.Header, expectedHeaders []TestHeader, t *testing.T) {
	for _, expected := range expectedHeaders {
		found := false
		for _, headerValue := range header[expected.key] {
			if strings.Contains(headerValue, expected.value) {
				found = true
			}
		}
		if !found {
			t.Errorf("Failed to find Header %s with value %s", expected.key, expected.value)
		}
	}
}

type TestHeader struct {
	key   string
	value string
}

func TestServeHTTPHandlesCORSWhenNotOptionsRequest(t *testing.T) {
	req, err := http.NewRequest("POST", "www.example.com", strings.NewReader("some data"))
	if err != nil {
		t.Error("Error creating request in TestServeHTTPHandlesCORSWhenNotOptionsRequest: ", err)
	}
	w := httptest.NewRecorder()
	wsh := newTestRoomHandler(t)
	wsh.ServeHTTP(w, req)
	expectedHeaders := []TestHeader{
		{"Access-Control-Allow-Origin", wsh.origin},
		{"Access-Control-Expose-Headers", "Success"},
		{"Access-Control-Expose-Headers", "Code"},
	}
	checkHeadersPresent(w.Header(), expectedHeaders, t)
}

func TestServeHTTPRegisterSuccess(t *testing.T) {
	req, err := http.NewRequest("POST", "www.example.com/register", strings.NewReader("[\"Bob\", \"BobsPassword\"]"))
	if err != nil {
		t.Error("Error creating request in TestServeHTTPRegisterSuccess: ", err)
	}
	w := httptest.NewRecorder()
	wsh := newTestRoomHandler(t)
	wsh.ServeHTTP(w, req)
	expectedHeaders := []TestHeader{
		{"Success", "true"},
	}
	checkHeadersPresent(w.Header(), expectedHeaders, t)
}

func TestServeHTTPRegisterClientExists(t *testing.T) {
	req, err := http.NewRequest("POST", "www.example.com/register", strings.NewReader("[\"Fred\", \"FredsPassword\"]"))
	if err != nil {
		t.Fatal("Error creating request in TestServeHTTPRegisterClientExists: ", err)
	}
	w := httptest.NewRecorder()
	wsh := newTestRoomHandler(t)
	wsh.ServeHTTP(w, req)
	expectedHeaders := []TestHeader{
		{"Success", "false"},
	}
	checkHeadersPresent(w.Header(), expectedHeaders, t)
}

func TestLoginfailure(t *testing.T) {
	req, err := http.NewRequest("POST", "www.example.com/login", strings.NewReader("[\"Bob\", \"FredsPassword\"]"))
	if err != nil {
		t.Fatal("Error creating request in TestLoginSuccess: ", err)
	}
	w := httptest.NewRecorder()
	wsh := newTestRoomHandler(t)
	wsh.ServeHTTP(w, req)
	expectedHeaders := []TestHeader{
		{"Success", "false"},
	}
	checkHeadersPresent(w.Header(), expectedHeaders, t)
}

func newTestRoomHandler(t *testing.T) *RoomHandler {
	factory, err := newTestMemDataFactory()
	if err != nil {
		t.Fatal("Error creating handler: ", err)
	}
	roomlist := room.NewRoomList(100)
	wsh := NewRoomHandler(Options{Origin: "test origin", ChatLog: new(bytes.Buffer), RoomList: roomlist, DataFactory: factory, ClientFactory: client.NewFactory(roomlist, new(bytes.Buffer), factory)})
	return wsh
}

func newTestMemDataFactory() (clientdata.Factory, error) {
	df := filedata.NewMemDataFactory()
	cd := df.Create("Fred")
	err := cd.NewClient("FredsPassword")
	return df, err
}
