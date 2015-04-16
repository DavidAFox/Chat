package testclient

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
)

//TelnetClient is a Client used for testing the server's telnet side.
type TelnetClient struct {
	name     string
	room     string
	conn     net.Conn
	test     *testing.T
	res      *Result
	messages []string
	ip       string
	port     string
}

//NewTelnetClient returns a new telnet client.
func NewTelnetClient(name string, ip string, port string, rh *ResultHandler, t *testing.T) *TelnetClient {
	cl := new(TelnetClient)
	cl.name = name
	cl.test = t
	cl.res = NewResult(name, rh)
	cl.ip = ip
	cl.port = port
	return cl
}

//Name returns the clients name.
func (cl TelnetClient) Name() string {
	return cl.name
}

//Login connects the client to the server and starts GetMessages, its messages retrieval method.
func (cl *TelnetClient) Login() {
	var err error
	cl.conn, err = net.Dial("tcp", net.JoinHostPort(cl.ip, cl.port))
	if err != nil {
		cl.test.Errorf("Telnet Login() Error connecting: %v", err)
	}
	if cl.conn == nil {
		panic("Error with login. No connection.")
	}
	_, err = io.WriteString(cl.conn, cl.name+"\n")
	if err != nil {
		cl.test.Errorf("Telnet Login() Error writing name:  %v", err)
	}
	go cl.GetMessages()
}

//Join adds the client to a room and updates its results.
func (cl *TelnetClient) Join(rm string) {
	_, err := io.WriteString(cl.conn, fmt.Sprintf("/join %v\n", rm))
	if err != nil {
		cl.test.Errorf("Telnet Join() Error writing join in Room:%v :%v", rm, err)
	}
	if rm == "" {
		cl.res.Add("Must enter a Room to join")
		cl.checkupdate()
		return
	}
	if cl.room != "" {
		cl.res.JoinSend(fmt.Sprintf("%v leaves the room.", cl.Name()))
		cl.checkupdate()
	}
	cl.room = rm
	cl.res.Join(rm)
	cl.res.JoinSend(fmt.Sprintf("%v has joined the room.", cl.Name()))
}

//Send transmits the message to the clients room and updates its results.
func (cl *TelnetClient) Send(m string) {
	if m[0] == '/' {
		cl.test.Errorf("Telnet Send() Error command in Send: %v", m)
		return
	}
	_, err := io.WriteString(cl.conn, m+"\n")
	if err != nil {
		cl.test.Errorf("Telnet Send() Error writing send: %v", err)
	}
	if cl.room == "" {
		cl.res.Add("You're not in a room.  Type /join roomname to join a room or /help for other commands.")
	} else {
		cl.res.Send(fmt.Sprintf("[%v]: %v", cl.Name(), m))
	}
}

//Block adds name to client's block list and updates its results.
func (cl *TelnetClient) Block(name string) {
	_, err := io.WriteString(cl.conn, fmt.Sprintf("/block %v\n", name))
	if err != nil {
		cl.test.Errorf("Telnet Block() Error writing: %v", err)
	}
	if name == cl.Name() {
		cl.res.Add("You can't block yourself.")
	} else {
		cl.res.Block(name)
		cl.res.Add(fmt.Sprintf("Now Blocking %v.", name))
	}
}

//UnBlock removes clients matching name from this client's block list and updates its results.
func (cl *TelnetClient) UnBlock(name string) {
	_, err := io.WriteString(cl.conn, fmt.Sprintf("/unblock %v\n", name))
	if err != nil {
		cl.test.Errorf("Telnet UnBlock() Error writing: %v", err)
	}
	if cl.res.UnBlock(name) {
		cl.res.Add(fmt.Sprintf("No longer blocking %v.", name))
	} else {
		cl.res.Add(fmt.Sprintf("You are not blocking %v.", name))
	}
}

//Who retrieves from the server a list of the clients currently in the room and updates its results.
func (cl *TelnetClient) Who(rm string) {
	_, err := io.WriteString(cl.conn, fmt.Sprintf("/who %v\n", rm))
	if err != nil {
		cl.test.Errorf("Telnet Who() Error writing: %v", err)
	}
	cl.res.Who(rm)
}

//List retrieves from the server a list of current rooms and updates its results.
func (cl *TelnetClient) List() {
	_, err := io.WriteString(cl.conn, "/list\n")
	if err != nil {
		cl.test.Errorf("Telnet List() Error writing: %v", err)
	}
	cl.res.List()
}

//GetMessages reads the messages from the server and stores them in the clients messages
func (cl *TelnetClient) GetMessages() {
	var err error = nil
	var m string
	s := bufio.NewReaderSize(cl.conn, 10000)
	for {
		m, err = readString(s)
		cl.messages = append(cl.messages, m)
		if err != nil {
			cl.test.Errorf("Telnet GetMessage() Error: %v", err)
		}
	}
}

//CheckResponse compares the clients messages recieved to those expected by its Results object and fails the test if they don't match.  It also panics to short circuit long tests when it goes out of sync.
func (cl *TelnetClient) CheckResponse() {
	cl.checkupdate()
	if len(cl.messages) > 0 {
		cl.messages[0] = strings.TrimPrefix(cl.messages[0], "What is your name? ") //removing What is your name? because it isn't followed by a newline
	}

	for i := range cl.messages {
		cl.messages[i] = RemoveTime(cl.messages[i])
	}
	if len(cl.res.Results) != len(cl.messages) {
		cl.test.Errorf("Telnet CheckResponse() Results and Messages len != Results: %v Messages:%v", len(cl.res.Results), len(cl.messages))
		if len(cl.messages) < len(cl.res.Results) {
			for x := range cl.messages {
				cl.test.Errorf("\nResult:  %v\nMessage: %v\n\n\n", cl.res.Results[x], cl.messages[x])
			}
			cl.test.Errorf("\nResult: %v", cl.res.Results[len(cl.res.Results)-1])
		} else {
			for x := range cl.res.Results {
				cl.test.Errorf("\nResult:  %v\nMessage: %v\n\n\n", cl.res.Results[x], cl.messages[x])
			}
			cl.test.Errorf("\nMessage: %v", cl.messages[len(cl.messages)-1])
		}
		cl.test.Errorf("Name: %v Result#: %v Messages#: %v", cl.Name(), len(cl.res.Results), len(cl.messages))
		cl.test.Errorf("\nClient: %v Room: %v\nIn Room: ", cl.name, cl.res.Room())
		for x := range cl.res.room.clients {
			cl.test.Errorf("\n%v", cl.res.room.clients[x].name)
		}
		cl.test.Errorf("Blocklist:")
		for x := range cl.res.blocklist {
			cl.test.Errorf("\n%v", cl.res.blocklist[x])
		}
		panic("Results # != Messages #")
	}
	for i := range cl.res.Results {
		if cl.res.Results[i] != RemoveTime(cl.messages[i]) {
			cl.test.Errorf("Telnet CheckResponse() got %v want %v.", RemoveTime(cl.messages[i]), cl.res.Results[i])
			cl.test.Errorf("Name: %v\nResults: %v\nMessages: %v\n", cl.name, cl.res.Results, cl.messages)
			for x := range cl.res.Results {
				cl.test.Errorf("\nResult:  %v\nMessage: %v\n\n\n", cl.res.Results[x], cl.messages[x])
			}
			cl.test.Errorf("\nClient: %v Room: %v\nIn Room: ", cl.name, cl.res.Room())
			for x := range cl.res.room.clients {
				cl.test.Errorf("\n%v", cl.res.room.clients[x].name)
			}
			panic("Results != Messages")
		}
	}
	cl.messages = cl.messages[:len(cl.messages)]
	cl.res.Results = cl.res.Results[:len(cl.res.Results)]
}

//Update is a method that runs checkupdate to match with the Client interface.
func (cl *TelnetClient) Update() {
	cl.checkupdate()
}

//checkupdate blocks until there are as many responses from the server as are expected by Results.  It doesn't check that they are the same only that the numbers match.
func (cl *TelnetClient) checkupdate() {
	start := time.Now()
	for len(cl.messages) < len(cl.res.Results) {
		time.Sleep(1 * time.Nanosecond)
		if time.Now().Sub(start) > 5*time.Second {
			cl.test.Errorf("Telnet GetMessage() timeout.\nName: %v\n\n\n\n\nMessages: %v\nResults: %v\n", cl.Name(), cl.messages, cl.res.Results)
			cl.test.Errorf("Messages: %v\nResults: %v\n", len(cl.messages), len(cl.res.Results))
			for x := range cl.messages {
				cl.test.Errorf("\nResult:  %v\nMessage: %v\n\n\n", cl.res.Results[x], cl.messages[x])
			}
			cl.test.Errorf("\nResult: %v", cl.res.Results[len(cl.res.Results)-1])
			panic("Timeout")
		}
	}
}

//readString reads a string from the connection ending with a '\n'and removes a '\r' if present.  readString also handles backspace characters in the stream.
func readString(conn io.Reader) (string, error) {
	r := make([]byte, 1)
	var ip string
	var err error
	_, err = conn.Read(r)
	for r[0] != '\n' {
		ip = ip + string(r[0])
		_, err = conn.Read(r)
	}
	if err != nil {
		fmt.Println(err)
	}
	re, err := regexp.Compile("[^\010]\010") //get rid of backspace and character in front of it
	if err != nil {
		fmt.Println("Error with regex in readString: ", err)
	}
	for re.MatchString(ip) { //keep getting rid of characters and backspaces as long as there are pairs left
		ip = re.ReplaceAllString(ip, "")
	}
	re2, err := regexp.Compile("^*\010") //get rid of any leading backspaces
	if err != nil {
		fmt.Println("Error with second regex in readString: ", err)
	}
	ip = re2.ReplaceAllString(ip, "")
	return strings.TrimSuffix(ip, "\r"), err
}
