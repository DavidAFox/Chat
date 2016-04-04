package chattest

import (
	"github.com/DavidAFox/Chat/testclient"
	"github.com/DavidAFox/gentest"
	"io"
	"os"
	"testing"
)

//Test represents a particular test.
type Test struct {
	httpIP        string
	httpPort      string
	telnetIP      string
	telnetPort    string
	httpClients   int
	telnetClients int
	restClients   int
	commands      int
	rooms         int
	test          *testing.T
	handler       *testclient.ResultHandler
}

//New creates a new test with default values.
func New(t *testing.T) *Test {
	tr := new(Test)
	tr.handler = testclient.NewResultHandler()
	tr.test = t
	tr.httpClients = 5
	tr.telnetClients = 5
	tr.restClients = 0
	tr.commands = 1000
	tr.rooms = 5
	tr.httpIP = "localhost"
	tr.httpPort = "8080"
	tr.telnetIP = "localhost"
	tr.telnetPort = "8000"
	return tr
}

//SetHTTPClients is for setting the number of HTTP clients to be used in the test.  The default number is 5.
func (ct *Test) SetHTTPClients(n int) {
	ct.httpClients = n
}

//SetTelnetClients is for setting the number of Telnet clients to be used in the test.  The default number is 5.
func (ct *Test) SetTelnetClients(n int) {
	ct.telnetClients = n
}

//SetRestClients is for setting the number of Rest clients to be used in the test. The default number is 1.
func (ct *Test) SetRestClients(n int) {
	ct.restClients = n
}

//SetCommands is for setting the number of client actions the test will run.  The default number is 100.
func (ct *Test) SetCommands(n int) {
	ct.commands = n
}

//SetRooms is for setting the number of chat rooms the test can create. The default number is 5.
func (ct *Test) SetRooms(n int) {
	ct.rooms = n
}

//SetHTTPIP is for setting the IP address that HTTP clients will try to connect to during the test.  The default is localhost.
func (ct *Test) SetHTTPIP(ip string) {
	ct.httpIP = ip
}

//SetHTTPPort is for setting the port that HTTP clients will try to connect to during the test.  The default is 8080.
func (ct *Test) SetHTTPPort(port string) {
	ct.httpPort = port
}

//SetTelnetIP is for setting the IP address that Telnet clients will try to connect to during the test.  The default is localhost.
func (ct *Test) SetTelnetIP(ip string) {
	ct.telnetIP = ip
}

//SetTelnetPort is for setting the port that Telnet Clients will try to connect to during the test.  The default is 8000.
func (ct *Test) SetTelnetPort(port string) {
	ct.telnetPort = port
}

//NewTelnetClient returns a new TestClient using telnet.
func (ct *Test) NewTelnetClient(name string) *testclient.TestClient {
	telnet := testclient.NewTelnetClient(name, ct.telnetIP, ct.telnetPort, ct.handler, ct.test)
	telnet.Login()
	cl := testclient.NewTestClient(telnet)
	return cl
}

//NewHTTPClient returns a new TestClient using HTTP.
func (ct *Test) NewHTTPClient(name string) *testclient.TestClient {
	http := testclient.NewHTTPClient(name, ct.httpIP, ct.httpPort, ct.handler, ct.test)
	http.Login()
	cl := testclient.NewTestClient(http)
	return cl
}

//NewRestClient returns a new TestClient using the REST interface.
func (ct *Test) NewRestClient(name string) *testclient.TestClient {
	rest := testclient.NewRestClient(name, ct.httpIP, ct.httpPort, ct.handler, ct.test)
	cl := testclient.NewTestRestClient(rest)
	return cl
}

//Run starts the test.
func (ct *Test) Run() {
	clients := make([]gentest.Actor, ct.httpClients+ct.telnetClients+ct.restClients, ct.httpClients+ct.telnetClients+ct.restClients)
	clientNames := make([]string, ct.httpClients+ct.telnetClients, ct.httpClients+ct.telnetClients)
	var name string
	for i := 0; i < ct.httpClients; i++ {
		name = testclient.RandomString(15)
		clientNames[i] = name
		clients[i] = ct.NewHTTPClient(name)
	}
	for i := 0; i < ct.telnetClients; i++ {
		name = testclient.RandomString(15)
		clientNames[i+ct.httpClients] = name
		clients[i+ct.httpClients] = ct.NewTelnetClient(name)
	}
	for i := 0; i < ct.restClients; i++ {
		name = testclient.RandomString(15)
		//		clientNames[i+ct.httpClients+ct.telnetClients] = name	//rest client names are not being added to the list of potential clients to block
		clients[i+ct.httpClients+ct.telnetClients] = ct.NewRestClient(name)
	}
	roomNames := make([]string, ct.rooms, ct.rooms)
	for i := 0; i < ct.rooms; i++ {
		roomNames[i] = testclient.RandomString(15)
	}
	shared := new(testclient.ClientData)
	shared.Rooms = roomNames
	shared.Names = clientNames
	for i := range clients {
		shared.Clients = append(shared.Clients, clients[i].(*testclient.TestClient).Client())
	}
	test := gentest.New("test", clients, shared)
	test.SetCommands(ct.commands)
	test.Run()
	f, err := os.Create("Test_Actions")
	if err != nil {
		panic("Error opening file")
	}
	actions := test.Actions()
	for i := range actions {
		_, err = io.WriteString(f, actions[i]+"\n")
		if err != nil {
			panic("Error writing to file")
		}
	}
	for i := range shared.Clients {
		shared.Clients[i].CheckResponse()
	}
}
