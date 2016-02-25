package main

import (
	"encoding/json"
	"fmt"
	//	"github.com/davidafox/chat/chattest"
	//	"github.com/davidafox/chat/clientdata"
	//	httpcon "github.com/davidafox/chat/connections/http"
	//	"github.com/davidafox/chat/room"
	//	"github.com/davidafox/chat/testclient/testclientdata"
	"net"
	//	"net/http"
	//	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

//Server settings for running the test.
const HTTPTestIP = "192.168.1.51"
const HTTPTestPort = "8080"
const TelnetTestIP = "localhost"
const TelnetTestPort = "8000"
const TestChatLog = "Chatlog_Test"

var TestConf config = config{ListeningIP: TelnetTestIP, ListeningPort: TelnetTestPort, HTTPListeningIP: HTTPTestIP, HTTPListeningPort: HTTPTestPort, LogFile: TestChatLog}

var TestMessages = []TestMessage{
	{"Bob", "Hello World"},
	{"Fred", "!(DKS)A(D"},
	{"12:30pm", "s7493*^&%%(^%"},
	{"Bob", "hi"},
	{"Bob", "hello?"},
	{"Fred", "$(#))@**!"},
}

type TestMessage struct {
	Name string
	Text string
}

func (t TestMessage) Result() string {
	return fmt.Sprintf("[%v]: %v", t.Name, t.Text)
}

func newTestMessage(name, message string) *TestMessage {
	msg := new(TestMessage)
	msg.Name = name
	msg.Text = message
	return msg
}

//TestConfigure creates a test file and then checks to see if configure loads it properly.
func TestConfigure(t *testing.T) {
	f, err := os.Create("Config_test")
	if err != nil {
		fmt.Println("Error creating file in TestConfigure: ", err)
	}
	enc := json.NewEncoder(f)
	fconf := config{ListeningIP: "192.168.1.54", ListeningPort: "8000", HTTPListeningIP: "129.124.12.1", HTTPListeningPort: "4004", LogFile: "Logfile"}
	err = enc.Encode(&fconf)
	if err != nil {
		fmt.Println("Error encoding in Testconfigure: ", err)
	}
	conf := configure("Config_test")
	if *conf != fconf {
		t.Errorf("configure() %v => %v, want %v", fconf, conf, fconf)
	}
}

/*
//NewTestHTTPServer sets up an http test server with the roomhandler and resthandler.
func NewTestHTTPServer(rooms *room.RoomList, chl *os.File, conf *config, df clientdata.Factory) *httptest.Server {
	m := http.NewServeMux()
	room := httpcon.NewRoomHandler(rooms, chl, df, "")
	m.Handle("/", room)
	rest := newRestHandler(rooms, chl)
	m.Handle("/rest/", rest)
	hs := httptest.NewUnstartedServer(m)
	hs.Start()
	return hs
}
*/
//getIPPort is a helper function that returns the ip and port out of a string of the form http://ip:port.
func getIPPort(h string) (string, string, error) {
	h = strings.TrimPrefix(h, "http://")
	return net.SplitHostPort(h)
}

/*
//TestHttpAndTelnet runs a test with http, telnet and rest clients.
func TestHTTPAndTelnet(t *testing.T) {
	rooms := room.NewRoomList()
	conf := &TestConf
	chl, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		panic(err)
	}
	df := new(testclientdata.Factory)
	ts := NewTelnetServer(rooms, chl, conf, df)
	hs := NewTestHTTPServer(rooms, chl, conf, df)
	ip, port, err := getIPPort(hs.URL)
	if err != nil {
		panic(err)
	}
	go ts.Start()
	te := chattest.New(t)
	te.SetHTTPIP(ip)
	te.SetHTTPPort(port)
	te.SetHTTPClients(10)
	te.SetTelnetClients(10)
	te.SetRestClients(2)
	te.SetCommands(10000)
	te.Run()
	ts.Stop()
	hs.Close()
}
*/
/*
//TestHTTP runs a test with only http clients.
func TestHTTP(t *testing.T) {
	rooms := room.NewRoomList()
	conf := &TestConf
	chl, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		panic(err)
	}
	df := new(testclientdata.Factory)
	hs := NewTestHTTPServer(rooms, chl, conf, df)
	ip, port, err := getIPPort(hs.URL)
	if err != nil {
		panic(err)
	}
	te := chattest.New(t)
	te.SetHTTPIP(ip)
	te.SetHTTPPort(port)
	te.SetTelnetClients(0)
	te.SetRestClients(0)
	te.SetHTTPClients(10)
	te.SetCommands(10000)
	te.Run()
	hs.Close()
}
*/
/*
//TestTelnet runs a test with only telnet clients.
func TestTelnet(t *testing.T) {
	rooms := room.NewRoomList()
	conf := &TestConf
	chl, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		panic(err)
	}
	df := new(testclientdata.Factory)
	ts := NewTelnetServer(rooms, chl, conf, df)
	go ts.Start()
	te := chattest.New(t)
	te.SetHTTPClients(0)
	te.SetTelnetClients(10)
	te.SetRestClients(0)
	te.SetCommands(10000)
	te.Run()
	ts.Stop()
}
*/
