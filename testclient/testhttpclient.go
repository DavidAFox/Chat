package testclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
)

//HTTPClient is a Client used for testing the server's HTTP side.
type HTTPClient struct {
	name     string
	token    string
	room     string
	client   *http.Client
	test     *testing.T
	res      *Result
	messages []string
	ip       string
	port     string
}

//NewHTTPClient returns a new http client.
func NewHTTPClient(name string, ip string, port string, rh *ResultHandler, t *testing.T) *HTTPClient {
	cl := new(HTTPClient)
	cl.name = name
	cl.client = &http.Client{}
	cl.test = t
	cl.res = NewResult(name, rh)
	cl.ip = ip
	cl.port = port
	return cl
}

//Name returns the name of the client.
func (cl *HTTPClient) Name() string {
	return cl.name
}

type Lgn struct {
	Name     string
	Password string
}

//Login connects the client to the server and sets the clients token for future requests.
func (cl *HTTPClient) Login() {
	l := &Lgn{Name: cl.Name(), Password: "a"}
	enc, err := json.Marshal(l)
	if err != nil {
		cl.test.Errorf("HTTP Login() Error encoding: %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/login", net.JoinHostPort(cl.ip, cl.port)), bytes.NewReader(enc))
	resp, err := cl.client.Do(req)
	if resp == nil || resp.Header.Get("success") == "false" {
		panic("Error with login.")
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&cl.token)
	if err != nil {
		cl.test.Errorf("HTTP Login() Error decoding token: %v", err)
	}
	resp.Body.Close()
}

//Join adds the client to a room and updates its results.
func (cl *HTTPClient) Join(rmName string) {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/rooms/%v/join", net.JoinHostPort(cl.ip, cl.port), rmName), nil)
	if err != nil {
		cl.test.Errorf("HTTP Join() Error making post request: %v", err)
	}
	if cl.token != "" {
		req.Header.Set("Authorization", cl.token)
	}
	resp, err := cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP Join() Error doing post: %v", err)
	}
	resp.Body.Close()
	cl.room = rmName
	if cl.res.room != nil {
		cl.res.JoinSend(fmt.Sprintf("%v leaves the room.", cl.Name()))
	}
	cl.res.Join(rmName)
	cl.res.JoinSend(fmt.Sprintf("%v has joined the room.", cl.Name()))
}

//Send transmits the message to the clients room and updates its results.
func (cl *HTTPClient) Send(msg string) {
	enc, err := json.Marshal(msg)
	if err != nil {
		cl.test.Errorf("HTTP Send() Error encoding message in Send: %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/messages", net.JoinHostPort(cl.ip, cl.port)), bytes.NewReader(enc))
	if err != nil {
		cl.test.Errorf("HTTP Send() Error making new request: %v", err)
	}
	req.Header.Set("Authorization", cl.token)
	resp, err := cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP Send() Error sending request: %v", err)
	}
	if cl.room != "" {
		if resp.StatusCode != 200 {
			cl.test.Errorf("HTTP Send() got: %v want: 200", resp.StatusCode)
		}
		cl.res.Send(fmt.Sprintf("[%v]: %v", cl.Name(), msg))
	} else {
		cl.res.Add("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
	}
}

//Block adds the name to client's block list and updates its results.
func (cl *HTTPClient) Block(name string) {
	enc, err := json.Marshal(name)
	if err != nil {
		cl.test.Errorf("HTTP Block() Error encodiong name: %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/block", net.JoinHostPort(cl.ip, cl.port)), bytes.NewReader(enc))
	if err != nil {
		cl.test.Errorf("HTTP Block() Error making request: %v", err)
	}
	req.Header.Set("Authorization", cl.token)
	_, err = cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP Block() Error sending request: %v", err)
	}
	if name == cl.Name() {
		cl.messages = append(cl.messages, "You can't block yourself.")
		cl.res.Add("You can't block yourself.")
	} else {
		cl.messages = append(cl.messages, fmt.Sprintf("Now Blocking %v.", name))
		cl.res.Block(name)
		cl.res.Add(fmt.Sprintf("Now Blocking %v.", name))
	}
}

//Unblock removes clients matching name from this client's block list and updates its results.
func (cl *HTTPClient) UnBlock(name string) {
	enc, err := json.Marshal(name)
	if err != nil {
		cl.test.Errorf("HTTP UnBlock() Error encoding name: %v", err)
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%v/unblock", net.JoinHostPort(cl.ip, cl.port)), bytes.NewReader(enc))
	if err != nil {
		cl.test.Errorf("HTTP UnBlock() Error making request: %v", err)
	}
	req.Header.Set("Authorization", cl.token)
	resp, err := cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP UnBlock() Error sending request: %v", err)
	}
	var found bool
	/*	dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&found)
		if err != nil {
			cl.test.Errorf("HTTP UnBlock() Error decoding: %v", err)
		}
	*/found = (resp.Header.Get("success") == "true")
	if found {
		cl.messages = append(cl.messages, fmt.Sprintf("No longer blocking %v.", name))
	}
	if !found {
		cl.messages = append(cl.messages, fmt.Sprintf("You are not blocking %v.", name))
	}
	if cl.res.UnBlock(name) {
		cl.res.Add(fmt.Sprintf("No longer blocking %v.", name))
	} else {
		cl.res.Add(fmt.Sprintf("You are not blocking %v.", name))
	}
}

//Who retrieves from the server a list of the clients currently in the room and updates its results.
func (cl *HTTPClient) Who(rmName string) {
	var req *http.Request
	var err error
	if rmName != "" {
		req, err = http.NewRequest("GET", fmt.Sprintf("http://%v/rooms/%v/who", net.JoinHostPort(cl.ip, cl.port), rmName), nil)
		if err != nil {
			cl.test.Errorf("HTTP Who() Error making new request: %v", err)
		}
	} else {
		req, err = http.NewRequest("GET", fmt.Sprintf("http://%v/rooms/who", net.JoinHostPort(cl.ip, cl.port)), nil)
		if err != nil {
			cl.test.Errorf("HTTP Who() Error making new request: %v", err)
		}
	}
	req.Header.Set("Authorization", cl.token)
	resp, err := cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP Who() Error sending request: %v", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		cl.messages = append(cl.messages, "Room not Found")
		cl.res.Add("Room not Found")
		resp.Body.Close()
		return
	}
	dec := json.NewDecoder(resp.Body)
	var room string
	err = dec.Decode(&room)
	if err != nil {
		cl.test.Errorf("HTTP Who() Error decoding room: %v", err)
	}
	clientlist := make([]string, 0, 0)
	err = dec.Decode(&clientlist)
	if err != nil {
		cl.test.Errorf("HTTP Who() Error decoding clientlist: %v", err)
	}
	cl.messages = append(cl.messages, fmt.Sprintf("Room: %v", room))
	for _, i := range clientlist {
		cl.messages = append(cl.messages, i)
	}
	resp.Body.Close()
	cl.res.Who(rmName)
}

//List retrieves from the server a list of current rooms and updates its results.
func (cl *HTTPClient) List() {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%v/rooms/list", net.JoinHostPort(cl.ip, cl.port)), nil)
	if err != nil {
		cl.test.Errorf("HTTP List() Error making request: %v", err)
	}
	req.Header.Set("Authorization", cl.token)
	resp, err := cl.client.Do(req)
	if err != nil {
		cl.test.Errorf("HTTP List() Error seding request: %v", err)
	}
	dec := json.NewDecoder(resp.Body)
	list := make([]string, 0, 0)
	err = dec.Decode(&list)
	if err != nil {
		cl.test.Errorf("HTTP List() Error decoding: %v", err)
	}
	cl.messages = append(cl.messages, "Rooms:")
	for _, i := range list {
		cl.messages = append(cl.messages, i)
	}
	resp.Body.Close()
	cl.res.List()
}

//GetMessages retrieves the clients current messages from the server.
func (cl *HTTPClient) GetMessages() {
	for len(cl.messages) < len(cl.res.Results) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%v/messages", net.JoinHostPort(cl.ip, cl.port)), nil)
		req.Header.Set("Authorization", cl.token)
		resp, err := cl.client.Do(req)
		if err != nil {
			cl.test.Errorf("HTTP GetMessages() Error sending request3: %v", err)
		}
		dec := json.NewDecoder(resp.Body)
		messages := make([]string, 0, 0)
		err = dec.Decode(&messages)
		if err != nil {
			cl.test.Errorf("HTTP GetMessages() Error decoding messages: %v", err)
		}
		resp.Body.Close()
		cl.messages = append(cl.messages, messages...)
	}
}

//CheckResponse compares the clients messages recieved to those expected by its Results object and fails the test if they don't match.  It also panics to short circuit long tests when it goes out of sync.
func (cl *HTTPClient) CheckResponse() {
	cl.GetMessages()
	for i := range cl.messages {
		cl.messages[i] = RemoveTime(cl.messages[i])
	}
	if len(cl.res.Results) != len(cl.messages) {
		cl.test.Errorf("Results # != Messages #")
		return
	}
	for i := range cl.res.Results {
		if cl.res.Results[i] != RemoveTime(cl.messages[i]) {
			cl.test.Errorf("HTTP CheckResponse got %v want %v.", RemoveTime(cl.messages[i]), cl.res.Results[i])
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

//Update runs GetMessage to update the client's messages from the server.
func (cl *HTTPClient) Update() {
	cl.GetMessages()
}
