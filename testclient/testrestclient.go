package testclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
)

//RestClient is a type of client for testing the servers REST api.
type RestClient struct {
	name     string
	messages []string
	client   *http.Client
	test     *testing.T
	res      *Result
	ip       string
	port     string
}

//NewRestClient returns a new RestClient.
func NewRestClient(name string, ip string, port string, rh *ResultHandler, t *testing.T) *RestClient {
	cl := new(RestClient)
	cl.name = name
	cl.client = &http.Client{}
	cl.messages = make([]string, 0, 0)
	cl.res = NewResult(name, rh)
	cl.test = t
	cl.ip = ip
	cl.port = port
	return cl
}

//Name returns the name of the client.
func (cl RestClient) Name() string {
	return cl.name
}

//Send sends the message to the room.
func (cl *RestClient) Send(m, room string) {
	resp, err := restPost(cl.name, m, room, cl.client, cl.ip, cl.port)
	if err != nil {
		cl.test.Errorf("Error with restPost: %v", err)
	}
	if resp.StatusCode == 404 { //room not found
		cl.messages = append(cl.messages, "Room not Found")
		cl.res.RestSend(m, room)
		return
	}
	if resp.StatusCode != 200 {
		cl.test.Errorf("Rest Send() Got %v want 200.  Client:%v Message:%v Room:%v", resp.StatusCode, cl.name, m, room)
	}
	cl.res.RestSend(fmt.Sprintf("[%v]: %v", cl.Name(), m), room)
	for !cl.checkupdate(m, room) {
	}
}

//checkupdate takes a message and a room and check to make sure that message was sent to that room.  It only checks the last message in the room.
func (cl *RestClient) checkupdate(m, room string) bool {
	resp, err := restGet(room, cl.client, cl.ip, cl.port)
	if err != nil {
		cl.test.Errorf("Error with restGet: %v", err)
	}
	if resp.StatusCode == 404 {
		return true
	}
	if resp.StatusCode != 200 {
		cl.test.Errorf("Rest checkupdate got %v want 200", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var messages []string
	err = dec.Decode(&messages)
	if err != nil {
		cl.test.Errorf("Error decoding in Rest checkupdate: %v", err)
	}
	resp.Body.Close()
	message := RemoveTime(messages[len(messages)-1])
	return message == fmt.Sprintf("[%v]: %v", cl.name, m)
}

//Get gets the messages from the room.
func (cl *RestClient) Get(room string) {
	resp, err := restGet(room, cl.client, cl.ip, cl.port)
	if err != nil {
		cl.test.Errorf("Error with restGet: %v", err)
	}
	if resp.StatusCode == 404 { //room not found
		cl.messages = append(cl.messages, "Room not Found")
		cl.res.RestGet(room)
		return
	}
	if resp.StatusCode != 200 {
		cl.test.Errorf("Rest Get got %v want 200", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var messages []string
	err = dec.Decode(&messages)
	if err != nil {
		cl.test.Errorf("Error decoding in Rest Get: %v", err)
	}
	resp.Body.Close()
	for _, mg := range messages {
		cl.messages = append(cl.messages, mg)
	}
	cl.res.RestGet(room)
}

//CheckResponse checks to see if the next message in the clients messages list is equal to m.
func (cl *RestClient) CheckResponse() {
	for i := range cl.messages {
		cl.messages[i] = RemoveTime(cl.messages[i])
	}
	if len(cl.res.Results) != len(cl.messages) {
		cl.test.Errorf("Results # != Messages #")
		return
	}
	for i := range cl.res.Results {
		if cl.res.Results[i] != cl.messages[i] {
			cl.test.Errorf("Rest CheckResponse got %v want %v.", cl.messages[i], cl.res.Results[i])
			cl.test.Errorf("Name: %v\nResults: %v\nMessages: %v\n", cl.name, cl.res.Results, cl.messages)
			for x := range cl.res.Results {
				cl.test.Errorf("\nResult:  %v\nMessage: %v\n\n\n", cl.res.Results[x], cl.messages[x])
			}
			cl.test.Errorf("\nResult: %v", cl.res.Results[len(cl.res.Results)-1])
			cl.test.Errorf("\nResult#: %v   Message#: %v\n", len(cl.res.Results), len(cl.messages))
			break
		}
	}
}

//Update is empty so that a rest client can meet the Client interface.  A rest client can not be updated because it doesn't expect anything specific from the server.
func (cl *RestClient) Update() {
}

//restPost is a helper function for creating a rest post request.
func restPost(name, message, room string, client *http.Client, ip string, port string) (*http.Response, error) {
	msg := newTestMessage(name, message)
	enc, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding in restPost: ", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/rest/%v", net.JoinHostPort(ip, port), room), bytes.NewReader(enc))
	if err != nil {
		fmt.Println("Error creating request in restPost: ", err)
	}
	return client.Do(req)
}

//restGet is a helper function for creating a rest get request.
func restGet(room string, client *http.Client, ip string, port string) (*http.Response, error) {
	return client.Get(fmt.Sprintf("http://%v/rest/%v", net.JoinHostPort(ip, port), room))
}
