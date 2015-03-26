package main

import (
	"testing"
	"os"
	"fmt"
	"encoding/json"
//	"regexp"
	"github.com/davidafox/chat/chattest"
)

const HTTPTestIP = "localhost"
const HTTPTestPort = "8080"
const TelnetTestIP = "localhost"
const TelnetTestPort = "8000"
const TestChatLog = "Chatlog_Test"
var TestConf config = config{TelnetTestIP, TelnetTestPort, HTTPTestIP, HTTPTestPort, TestChatLog}

var TestMessages = []TestMessage {
	{"Bob","Hello World"},
	{"Fred","!(DKS)A(D"},
	{"12:30pm", "s7493*^&%%(^%"},
	{"Bob", "hi"},
	{"Bob", "hello?"},
	{"Fred","$(#))@**!"},
}

type TestMessage struct {
	Name string
	Text string
}

func (t TestMessage) Result() string {
	return fmt.Sprintf("[%v]: %v", t.Name, t.Text)
}

//TestConfigure creates a test file and then checks to see if configure loads it properly.
func TestConfigure(t *testing.T) {
	f, err := os.Create("Config_test")
	if err != nil {
		fmt.Println("Error creating file in TestConfigure: ", err)
	}
	enc := json.NewEncoder(f)
	fconf := config{"192.168.1.54","8000", "129.124.12.1","4004","Logfile"}
	err = enc.Encode(&fconf)
	if err != nil {
		fmt.Println("Error encoding in Testconfigure: ", err)
	}
	conf := configure ("Config_test")
	if *conf != fconf {
		t.Errorf("configure() %v => %v, want %v",fconf,conf,fconf)
	}
}


func newTestMessage (name, message string) *TestMessage {
	msg := new(TestMessage)
	msg.Name = name
	msg.Text = message
	return msg
}
/*
func TestResult(t *testing.T){
	te := chattest.NewChatTest(t)
	res1 := te.NewResult("test1")
	res2 := te.NewResult("test2")
	res3 := te.NewResult("test3")
	res1.Join("Room")
	res2.Join("Room")
	res3.Join("Room3")
	res1.Block("test3")
	res1.Send("1m1")
	res1.Send("1m2")
	res2.Send("2m3")
	res3.Send("3m4")
	res3.Join("Room")
	res3.Send("3m5")
	res1.UnBlock("test3")
	res3.Send("3m6")
	res1.Send("1m7")
	res1.Join("Room3")
	res1.Send("1m8")
	res2.Send("2m9")
//	fmt.Printf("res1: %v\n",res1.Results)
//	fmt.Printf("res2: %v\n",res2.Results)
//	fmt.Printf("res3: %v\n",res3.Results)
}
*/
/*
func TestRestHandler(t *testing.T){
	rooms := NewRoomList()
	rooms.Add(NewRoom("Room"))
	conf := &TestConf
	chl, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
//	defer chl.Close()
	if err != nil {
		panic(err)
	}
	go server(rooms, chl, conf, make(chan bool, 1))
	go serverHTTP(rooms,chl, conf)
	clMap := make(map[string]*chattest.RestClient)
	for _, m := range TestMessages {
		clMap[m.Name] = chattest.NewRestClient(m.Name, t)
		clMap[m.Name].Join("Room")
	}
	for _, m := range TestMessages {
		clMap[m.Name].Send(m.Text)
	}
	for _, m := range TestMessages {
		for _, x := range clMap {
			x.CheckResponse(m.Result())
		}
	}
}
*/
/*
func TestTelnet(t *testing.T){
	te := chattest.NewChatTest(t)
	clMap := make(map[string]*chattest.TelnetClient)
	for _, m := range TestMessages {
		if _,ok := clMap[m.Name];!ok{
			clMap[m.Name] = te.NewTelnetClient(m.Name)
			clMap[m.Name].Login()
			clMap[m.Name].Join("Room5")
		}
	}
	for _, m := range TestMessages {
		clMap[m.Name].Send(m.Text)
	}
	for _,x := range clMap {
		x.CheckResponse()
	}
}
*/
func TestHTTP(t *testing.T){
	rooms := NewRoomList()
	conf := &TestConf
	chl, err := os.OpenFile(conf.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		panic(err)
	}
	ts := NewTelnetServer(rooms,chl,conf)
	go ts.Start()
	go serverHTTP(rooms,chl, conf)
	te := chattest.New(t)
	te.SetHTTPClients(10)
	te.SetTelnetClients(10)
	te.SetCommands(1000)
	te.Run()
/*	clMap := make(map[string]*chattest.HTTPClient)
	for _, m := range TestMessages {
		clMap[m.Name] = te.NewHTTPClient(m.Name)
		clMap[m.Name].Login()
		clMap[m.Name].Join("Room")
	}
	for _, m := range TestMessages {
		clMap[m.Name].Send(m.Text)
	}
	for _, x := range clMap {
		x.CheckResponse()
	}
*/
	ts.Stop()
}

type TestClient interface {
	CheckResponse(string)
	Send(string)
	Join(string)
	Name() string
}
/*
//removes the time and following space from a string if its present
func RemoveTime(s string) string {
	reg := regexp.MustCompile("\\A\\b\\d?\\d:\\d\\d[a,p]m\\b")
	time := reg.FindString(s)
	return s[len(time)+1:]
}
*/


