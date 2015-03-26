package chattest

import (
	"net/http"
	"fmt"
	"encoding/json"
	"bytes"
	"testing"
	"net"
)

//Not yet Implemented to work with the ChatTest object


//RestClient is a type of client for testing the servers REST api.
type RestClient struct {
	name string
	room string
	messages []string
	client *http.Client
	test *testing.T
	ip string
	port string
}

//NewRestClient returns a new RestClient.
func (ct *ChatTest) NewRestClient(name string, t *testing.T) *RestClient{
	cl := new(RestClient)
	cl.name = name
	cl.client = &http.Client{}
	cl.messages = make([]string,0,0)
	cl.test = ct.test
	cl.ip = ct.httpIP
	cl.port = ct.httpPort
	return cl
}

//Name returns the name of the client.
func (cl RestClient) Name() string {
	return cl.name
}

//Join sets the clients room to the rm.  Unlike the other test client join functions it does not interact with the server.
func (cl *RestClient) Join(rm string) {
	cl.room = rm
}

//Send sends the message to the client's room.
func (cl RestClient) Send(m string) {
	resp, err := restPost(cl.name,m, cl.room,cl.client, cl.ip, cl.port)
	if err != nil {
		cl.test.Errorf("Error with restPost: %v", err)
	}
	if resp.StatusCode != 200 {
		cl.test.Errorf("Rest Send() Got %v want 200.  Client:%v Message:%v Room:%v", resp.StatusCode, cl.name, m, cl.room)
	}
}

//CheckResponse checks to see if the next message in the clients messages list is equal to m.
func (cl *RestClient) CheckResponse(m string) {
	resp, err := restGet("Room", cl.client, cl.ip, cl.port)
	if err != nil {
		cl.test.Errorf("Error with restGet: %v", err)
	}
	if resp.StatusCode != 200 {
		cl.test.Errorf("Rest CheckResponse() rest GET got %v want 200", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	var messages []string
	err = dec.Decode(&messages)
	if err != nil {
		cl.test.Errorf("Error decoding in Rest CheckResponse: %v", err)
	}
	resp.Body.Close()
	for _, mg := range messages {
		cl.messages = append(cl.messages, mg)
	}
	newMessage := RemoveTime(cl.messages[0])
	cl.messages = cl.messages[1:]
	if newMessage != m {
		cl.test.Errorf("Rest CheckResponse() got %v want %v Client:%v Room:%v", newMessage, m,cl.name, cl.room)
	}
}

//restPost is a helper function for creating a rest post request.
func restPost(name,message,room string,client *http.Client, ip string, port string) (*http.Response, error) {
	msg := newTestMessage(name, message)
	enc, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error encoding in restPost: ", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/rest/%v", net.JoinHostPort(ip,port),room),bytes.NewReader(enc))
	if err != nil {
		fmt.Println("Error creating request in restPost: ", err)
	}
	return client.Do(req)
}

//restGet is a helper function for creating a rest get request.
func restGet(room string, client *http.Client, ip string, port string) (*http.Response, error) {
	return client.Get(fmt.Sprintf("http://%v/rest/%v", net.JoinHostPort(ip, port), room))
}

