package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/clientdata/filedata"
	"github.com/davidafox/chat/connections"
	"github.com/davidafox/chat/message"
	"github.com/davidafox/chat/room"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestHandleCommandNoArgs(t *testing.T) {
	conn, _ := newConnectionForTesting()
	_ = conn.HandleCommand(newTestCommand("test"))
	command := conn.client.(*testClient).lastCommand
	compareCommands([]string{"test"}, command, t)
}

func TestHandleCommandOneArgs(t *testing.T) {
	conn, _ := newConnectionForTesting()
	_ = conn.HandleCommand(newTestCommand("test", "arg1"))
	command := conn.client.(*testClient).lastCommand
	compareCommands([]string{"test", "arg1"}, command, t)
}

func TestHandleCommandMultipleArgs(t *testing.T) {
	conn, _ := newConnectionForTesting()
	_ = conn.HandleCommand(newTestCommand("test", "arg1", "arg2"))
	command := conn.client.(*testClient).lastCommand
	compareCommands([]string{"test", "arg1", "arg2"}, command, t)
}

func TestHandleSendCommand(t *testing.T) {
	conn, _ := newConnectionForTesting()
	_ = conn.HandleCommand(newTestCommand("send", "Hello", "World"))
	command := conn.client.(*testClient).lastCommand
	compareCommands([]string{"send", "Hello", "World"}, command, t)
}

func TestHandleQuit(t *testing.T) {
	conn, _ := newConnectionForTesting()
	_ = conn.HandleCommand(newTestCommand("quit"))
	client := conn.client.(*testClient)
	if client.leaveRoomCalled != 1 {
		t.Error("LeaveRoom called incorrect number of times expected: 1 got: ", client.leaveRoomCalled)
	}

}

func TestGettingMessage(t *testing.T) {
	conn, client := newConnectionForTesting()
	conn.SendMessage(message.NewServerMessage("test"))
	<-client.read
	conn.socket.Close()
	messages := client.messages
	expected := "Text: test"
	if len(messages) != 1 {
		t.Errorf("Incorrect number of messages expect: %v, got: %v", expected, client.messages)
	} else {
		if strings.Contains(messages[0], expected) {
			t.Errorf("Incorrect message expected: %s, got: %s", expected, messages[0])
		}
	}
}

func TestRegister(t *testing.T) {
	server, client := NewTestSocket()
	go Start(server, NewOptionsForTesting())
	go client.WriteMessage(TEXT_MESSAGE, []byte(newTestCommandString(t, "register", "Bob", "BobsPassword")))
	<-client.read
	if !strings.Contains(client.messages[0], "Account Created") {
		t.Error("Account Created message not found got: ", client.messages)
	}
	client.Close()
}

func TestLogin(t *testing.T) {
	server, client := NewTestSocket()
	go Start(server, NewOptionsForTesting())
	go client.WriteMessage(TEXT_MESSAGE, []byte(newTestCommandString(t, "login", "Fred", "FredsPassword")))
	<-client.read
	if !strings.Contains(client.messages[0], "Welcome") {
		t.Error("Welcome message not found got: ", client.messages)
	}
	client.Close()
}

func sliceContains(slice []string, str string) bool {
	for i := range slice {
		if slice[i] == str {
			return true
		}
	}
	return false
}

func compareCommands(expected, got []string, t *testing.T) {
	if len(expected) != len(got) {
		t.Errorf("Incorrect length of command expected: %v, got: %v", expected, got)
	} else {
		for i, _ := range expected {
			if expected[i] != got[i] {
				t.Errorf("Incorrect command expected: %v, got: %v", expected, got)
			}
		}
	}
}

func newConnectionForTesting() (*Connection, *testSocket) {
	tcf := new(testClientFactory)
	con := new(Connection)
	con.writeLock = new(sync.Mutex)
	serverSocket, clientSocket := NewTestSocket()
	con.socket = serverSocket
	con.client = tcf.New("testClient", con)
	return con, clientSocket
}

type testClientFactory struct {
}

func (cf *testClientFactory) New(name string, connection connections.Connection) connections.Client {
	cl := new(testClient)
	cl.connection = connection
	cl.name = name
	cl.executeCalled = 0
	cl.leaveRoomCalled = 0
	cl.lastCommand = []string{}
	cl.response = new(testResponse)
	return cl
}

type testClient struct {
	connection      connections.Connection
	name            string
	executeCalled   int
	leaveRoomCalled int
	lastCommand     []string
	response        *testResponse
}

func (cl *testClient) Execute(command []string) connections.Response {
	cl.executeCalled++
	cl.lastCommand = command
	if len(command) > 0 && command[0] == "quit" {
		cl.connection.Close()
		return cl.response
	}
	return cl.response
}

func (cl *testClient) LeaveRoom() {
	cl.leaveRoomCalled++
}

func (cl *testClient) Name() string {
	return cl.name
}

func (cl *testClient) SetConnection(conn connections.Connection) {
	cl.connection = conn
}

type testResponse struct {
	success bool
	code    int
	str     string
	data    interface{}
}

func (tr *testResponse) Success() bool {
	return tr.success
}

func (tr *testResponse) Code() int {
	return tr.code
}

func (tr *testResponse) String() string {
	return tr.str
}

func (tr *testResponse) Data() interface{} {
	return tr.data
}

type testSocket struct {
	reader   io.ReadCloser
	writer   io.WriteCloser
	messages []string
	read     chan (bool)
}

//NewTestSocket makes a pipe connecting the server and client sockets and meeting the socket interface.  It ignores the messageType always returning 1 in the ReadMessage unless there is an error.
func NewTestSocket() (server *testSocket, client *testSocket) {
	ts := new(testSocket)
	tc := new(testSocket)
	ts.reader, tc.writer = io.Pipe()
	tc.reader, ts.writer = io.Pipe()
	ts.read = make(chan (bool))
	tc.read = make(chan (bool))
	tc.messages = make([]string, 0, 1)
	ts.messages = make([]string, 0, 1)
	go tc.getMessages()
	return ts, tc
}

func (ts *testSocket) ReadMessage() (messageType int, p []byte, err error) {
	readBuff := make([]byte, 1024)
	var length int
	for !bytes.Contains(readBuff, []byte("\n")) {
		length, err = ts.reader.Read(readBuff)
		if err != nil && err != io.EOF {
			return 0, nil, err
		}
	}
	readBuff = readBuff[:length]
	readBuff = bytes.TrimSuffix(readBuff, []byte("\n"))
	return 1, readBuff, nil
}

func (ts *testSocket) WriteMessage(messageType int, p []byte) (err error) {
	p = append(p, []byte("\n")...)
	_, err = ts.writer.Write(p)
	return err
}

func (ts *testSocket) Close() error {
	ts.reader.Close()
	ts.writer.Close()
	return nil
}

func (cl *testSocket) getMessages() {
	for {
		_, m, err := cl.ReadMessage()
		if err != nil && err != io.EOF {
			return
		}
		if len(m) > 0 {
			cl.messages = append(cl.messages, string(m))
			cl.read <- true
		}
	}
}

func NewOptionsForTesting() *Options {
	o := new(Options)
	o.ClientFactory = new(testClientFactory)
	o.ChatLog = bytes.NewBufferString("")
	o.DataFactory, _ = newTestMemDataFactory()
	o.RoomList = room.NewRoomList(100)
	return o
}

func newTestMemDataFactory() (clientdata.Factory, error) {
	df := filedata.NewMemDataFactory()
	cd := df.Create("Fred")
	err := cd.NewClient("FredsPassword")
	return df, err
}

func newTestCommand(cmd string, args ...string) *Input {
	in := new(Input)
	in.Command = cmd
	in.Args = args
	return in
}

func newTestCommandString(t *testing.T, cmd string, args ...string) string {
	in := newTestCommand(cmd, args...)
	j, err := json.Marshal(in)
	if err != nil {
		t.Error("Error mashaling Command: ", err)
	}
	return string(j)
}
