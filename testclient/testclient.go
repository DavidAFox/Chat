package testclient

import (
	"fmt"
	"github.com/DavidAFox/gentest"
	"math/rand"
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
	Name() string
	Update()
}

//SharedClient is an interface for use in the shared ClientData
type SharedClient interface {
	Update()
	CheckResponse()
}

//join is an actor to do the clients join function.
type join struct {
	cl Client
}

func (j *join) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("join shared not ClientData")
	}
	room := randomName(data.Rooms)
	j.cl.Join(room)
	return fmt.Sprintf("Joined %v", room)
}

//send is an actor to do the clients send function.
type send struct {
	cl Client
}

func (s *send) Do(shared interface{}) string {
	message := RandomString(25)
	s.cl.Send(message)
	return fmt.Sprintf("Sent %v", message)
}

//block is an actor to do the clients block function.
type block struct {
	cl Client
}

func (b *block) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("block shared not ClientData")
	}
	name := randomName(data.Names)
	b.cl.Block(name)
	return fmt.Sprintf("blocked %v", name)
}

//unblock is an actor to do the clients unblock function.
type unblock struct {
	cl Client
}

func (u *unblock) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("unblock shared not ClientData")
	}
	name := randomName(data.Names)
	u.cl.UnBlock(name)
	return fmt.Sprintf("unblocked %v", name)
}

//list is an actor to do the clients list function.
type list struct {
	cl Client
}

func (l *list) Do(shared interface{}) string {
	l.cl.List()
	return fmt.Sprintf("list")
}

//who is an actor to do the clients who function.
type who struct {
	cl Client
}

func (w *who) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("who shared not ClientData")
	}
	room := randomName(data.Rooms)
	w.cl.Who(room)
	return fmt.Sprintf("who %v", room)
}

//NewActionList creates a slice of the clients actors.
func NewActionList(cl Client) []gentest.Actor {
	alist := make([]gentest.Actor, 6, 6)
	join := new(join)
	join.cl = cl
	alist[0] = join
	send := new(send)
	send.cl = cl
	alist[1] = send
	block := new(block)
	block.cl = cl
	alist[2] = block
	unblock := new(unblock)
	unblock.cl = cl
	alist[3] = unblock
	list := new(list)
	list.cl = cl
	alist[4] = list
	who := new(who)
	who.cl = cl
	alist[5] = who
	return alist
}

//restsend is an actor for a rest client's send function.
type restsend struct {
	cl *RestClient
}

func (rs *restsend) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("restsend shared not ClientData")
	}
	room := randomName(data.Rooms)
	msg := RandomString(25)
	rs.cl.Send(msg, room)
	return fmt.Sprintf("restsend %v to %v", msg, room)
}

//restget is an actor for a rest client's get function.
type restget struct {
	cl *RestClient
}

func (rg *restget) Do(shared interface{}) string {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("restget shared not ClientData")
	}
	room := randomName(data.Rooms)
	rg.cl.Get(room)
	return fmt.Sprintf("restget from %v", room)
}

//NewRestActionList creates a slice of rest client actors.
func NewRestActionList(cl *RestClient) []gentest.Actor {
	alist := make([]gentest.Actor, 2, 2)
	restsend := new(restsend)
	restsend.cl = cl
	alist[0] = restsend
	restget := new(restget)
	restget.cl = cl
	alist[1] = restget
	return alist
}

//TestClient can hold either a regular client or a rest client.  The client that it uses is determined by its action.
type TestClient struct {
	client Client
	rest   *RestClient
	action *gentest.Test
	name   string
}

//ClientData is an object for passing the necessary data into the actions.
type ClientData struct {
	Rooms   []string
	Names   []string
	Clients []SharedClient
}

//NewTestClient returns a test client with default equal weights for its actions.
func NewTestClient(cl Client) *TestClient {
	tcl := new(TestClient)
	tcl.client = cl
	tcl.name = cl.Name()
	tcl.action = gentest.New(cl.Name(), NewActionList(cl), cl)
	return tcl
}

//NewTestRestClient returns a test client that uses the rest inteface with default weights for its actions.
func NewTestRestClient(cl *RestClient) *TestClient {
	tcl := new(TestClient)
	tcl.rest = cl
	tcl.name = cl.Name()
	tcl.action = gentest.New(cl.Name(), NewRestActionList(cl), cl)
	return tcl
}

//NewTestClientWithWeights returns a test client with weighted action probabilites.  The weights slice should contain int for the weights in the order of join, send, block, unblock, list, who.
func NewTestClientWithWeights(cl Client, weights []int) *TestClient {
	tcl := new(TestClient)
	tcl.client = cl
	tcl.name = cl.Name()
	tcl.action = gentest.NewWithWeights(cl.Name(), NewActionList(cl), cl, weights)
	return tcl
}

//Client() returns the TestClient's Client.
func (tc *TestClient) Client() SharedClient {
	if tc.client != nil {
		return tc.client
	}
	if tc.rest != nil {
		return tc.rest
	}
	return nil
}

//Do is a method that allows the test client to meet the Actor interface.  It will run a random client action from its action list.
func (tc *TestClient) Do(shared interface{}) string {
	str := tc.action.Do(shared)
	update(shared)
	return fmt.Sprintf("%v", str)
}

//update takes a ClientData in interface{} form and runs Update() on each of the cleitns in the client list.
func update(shared interface{}) {
	var data *ClientData
	var ok bool
	if data, ok = shared.(*ClientData); !ok {
		panic("shared not ClientData")
	}
	for i := range data.Clients {
		data.Clients[i].Update()
	}
}

//RandomString returns a random string of maxlen > len > 0 made up of upper and lower case letters and/or numbers.
func RandomString(maxlen int) string {
	x := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	l := rand.Int()%maxlen + 1
	var s string
	for i := 0; i < l; i++ {
		s = s + string(x[rand.Int()%len(x)])
	}
	return s
}

//randomName is a helper function that returns a random room name from rml.
func randomName(nl []string) string {
	return nl[rand.Int()%len(nl)]
}
