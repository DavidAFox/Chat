//Package chattest is for testing the Chat chat server program.  A ChatTest can be created and configured to generate clients and have them perform random interactions with the server.  The Result object will track the responses the Client should get from the server and the CheckResponse function of each client can be used to compare the actual responses the client recieved to those expected by Result.
package chattest

import (
	"testing"
	"math/rand"
	"time"
)

//Client is an interface for using the ChatTest including all the functions that it calls
type Client interface {
	Login()
	Block(string)
	UnBlock(string)
	Who(string)
	List()
	Join(string)
	Send(string)
	CheckResponse()
	Name()string
	Update()
}

//ChatTest is an object representing a single test.  It starts with default values that can be changed with its set methods before calling run to start the test.
type ChatTest struct {
	join chan *joinrq
	test *testing.T
	list chan chan []string
	httpClients int
	telnetClients int
	commands int
	rooms int
	httpIP string
	httpPort string
	telnetIP string
	telnetPort string
}

//New creates a new chattest with default values that can be changed using the set methods.
func New(t *testing.T) *ChatTest {
	tr := new(ChatTest)
	tr.join = make(chan *joinrq)
	tr.test = t
	tr.list = make(chan chan []string)
	tr.httpClients = 5
	tr.telnetClients = 5
	tr.commands = 100
	tr.rooms = 5
	tr.httpIP = "localhost"
	tr.httpPort = "8080"
	tr.telnetIP = "localhost"
	tr.telnetPort = "8000"
	go roomManager(tr.join,tr.list)
	return tr
}

//SetHTTPClients is for setting the number of HTTP clients to be used in the test.  The default number is 5.
func (ct *ChatTest) SetHTTPClients(n int) {
	ct.httpClients = n
}

//SetTelnetClients is for setting the number of Telnet clients to be used in the test.  The default number is 5.
func (ct *ChatTest) SetTelnetClients(n int) {
	ct.telnetClients = n
}

//SetCommands is for setting the number of client actions the test will run.  The default number is 100.
func (ct *ChatTest) SetCommands(n int) {
	ct.commands = n
}

//SetRooms is for setting the number of chat rooms the test can create. The default number is 5.
func (ct *ChatTest) SetRooms(n int) {
	ct.rooms = n
}

//SetHTTPIP is for setting the IP address that HTTP clients will try to connect to during the test.  The default is localhost.
func (ct *ChatTest) SetHTTPIP(ip string) {
	ct.httpIP = ip
}

//SetHTTPPort is for setting the port that HTTP clients will try to connect to during the test.  The default is 8080.
func (ct *ChatTest) SetHTTPPort(port string) {
	ct.httpPort = port
}

//SetTelnetIP is for setting the IP address that Telnet clients will try to connect to during the test.  The default is localhost.
func (ct *ChatTest) SetTelnetIP(ip string) {
	ct.telnetIP = ip
}

//SetTelnetPort is for setting the port that Telnet Clients will try to connect to during the test.  The default is 8000.
func (ct *ChatTest) SetTelnetPort(port string) {
	ct.telnetPort = port
}


//NewTelnetClient creates and initializes a new telnet client.
func (tr *ChatTest) NewTelnetClient(name string) *TelnetClient {
	cl := new(TelnetClient)
	cl.name = name
	cl.messages = make([]string, 0, 10)
	cl.test = tr.test
	cl.res = tr.NewResult(name)
	cl.ip = tr.telnetIP
	cl.port = tr.telnetPort
	return cl
}

//NewHTTPClient creates and initializes a new http client.
func (tr *ChatTest) NewHTTPClient(name string) *HTTPClient{
	cl := new(HTTPClient)
	cl.name = name
	cl.client = &http.Client{}
	cl.test = tr.test
	cl.res = tr.NewResult(name)
	cl.ip = tr.httpIP
	cl.port = tr.httpPort
	return cl
}

//Run starts the test.  It should be called after the test is configured with the set methods.
func (ct *ChatTest) Run() {
	clients := make([]Client,ct.httpClients+ct.telnetClients,ct.httpClients+ct.telnetClients)
	rand.Seed(int64(time.Now().Nanosecond()))
	for i := 0; i<ct.httpClients;i++{
		clients[i] = ct.NewHTTPClient(randomString(15))
	}
	for i:= 0; i<ct.telnetClients;i++{
		clients[i+ct.httpClients] =  ct.NewTelnetClient(randomString(15))
	}
	rooms := make([]string, ct.rooms, ct.rooms)
	for i :=0; i<ct.rooms;i++{
		rooms[i] = randomString(15)
	}
	for i := range clients {
		clients[i].Login()
		clients[i].Join(rooms[0])
	}
	for i:= 0; i<ct.commands;i++{
		updateClients(clients)
/*		if i % 100 == 0 {//used to check if the Result and messages don't match during the test so you can stop long tests early if they get out of sync
			for x := range clients {
				clients[x].CheckResponse()
			}
		}
*/		switch rand.Int() % 6 {
			case 0:
				clients[rand.Int() % len(clients)].Join(randomRoom(rooms))
			case 1:
				clients[rand.Int() % len(clients)].Block(clients[rand.Int()%len(clients)].Name())
			case 2:
				clients[rand.Int() % len(clients)].UnBlock(clients[rand.Int()%len(clients)].Name())
			case 3:
				clients[rand.Int() % len(clients)].Who(randomRoom(rooms))
			case 4:
				clients[rand.Int() % len(clients)].List()
			case 5:
				clients[rand.Int() % len(clients)].Send(randomString(100))
		}
	}
	for i:= range clients {
		clients[i].CheckResponse()
	}
}

//randomString is a helper function that returns a random string of numbers or letters with a random length from 1 to maxlen.
func randomString(maxlen int) string {
	x:= []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	l:= rand.Int() % maxlen + 1
	var s string
	for i:=0;i<l;i++{
		s= s+string(x[rand.Int() % len(x)])
	}
	return s
}

//randomRoom is a helper function that returns a random room name from rml.
func randomRoom(rml []string) string {
	return rml[rand.Int() % len(rml)]
}

//updateClients is a helper function that runs Update() on each of the clients in the slice.
func updateClients(cl []Client){
	for i := range cl {
		cl[i].Update()
	}
}
