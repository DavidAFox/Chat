package main
/* A Chat Server */
import (
	"net"
	"io"
	"time"
	"encoding/json"
	"os"
	"log"
	"net/http"
	"os/signal"
)

//Client interface for working with the Room type.
type Client interface {
	Equals (other Client) bool
	Name() string
	Recieve(m Message)
}

//Message is an interface for dealing with various types of messages.
type Message interface {
	String() string
}

//config stores the configuration data from the config file.
type config struct {
	ListeningIP string
	ListeningPort string
	HTTPListeningIP string
	HTTPListeningPort string
	LogFile string
}



//configure loads the config file.
func configure (filename string) (c *config) {
	file, err := os.Open(filename)
	if err != nil {
		log.Panic("Error opening config file",err)
	}
	dec := json.NewDecoder(file)
	c = new(config)
	err = dec.Decode(c)
	if err !=nil {
		log.Panic("Error decoding config file",err)
	}
	return
}


type telnetServer struct {
	rooms *RoomList
	chatlog *os.File
	cls chan bool
	ln net.Listener
	done bool
}

func NewTelnetServer (rooms *RoomList, chl *os.File, c *config) *telnetServer {
	ts := new(telnetServer)
	var err error
	ts.ln, err = net.Listen("tcp", net.JoinHostPort(c.ListeningIP, c.ListeningPort))
	if err != nil {
		log.Panic(err)
	}
	ts.cls = make(chan bool, 1)
	ts.rooms = rooms
	ts.chatlog = chl
	ts.done = false
	return ts
}

func (ts *telnetServer) Stop(){
	ts.done = true
	ts.cls<-true
	ts.ln.Close()
}

//server listens for connections and sends them to handleConnection().
func (ts *telnetServer) Start () {
Outerloop:
	for{
		select {
			case <-ts.cls:
				break Outerloop
			default:
				conn, err := ts.ln.Accept()
				if err != nil && ts.done == false {
					log.Println(err)
				}
				if conn != nil {
					go handleConnection(conn,ts.rooms,ts.chatlog)
				}
		}
	}
}

//serverHTTP sets up the http handlers and then runs ListenAndServe
func serverHTTP (rooms *RoomList, chl *os.File, c *config) {
	room := newRoomHandler(rooms,chl)
	http.Handle("/", room)
	rest := newRestHandler(rooms,chl)
	http.Handle("/rest/", rest)
	err := http.ListenAndServe(net.JoinHostPort(c.HTTPListeningIP,c.HTTPListeningPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

//restHandler is the http.Handler for handling the REST API
type restHandler struct {
	rooms *RoomList
	chl *os.File
}

//newRestHandler initializes a new restHandler.
func newRestHandler(rooms *RoomList, chl *os.File) *restHandler{
	m := new(restHandler)
	m.rooms = rooms
	m.chl = chl
	return m
}

//ServeHTTP handles the restServer's http requests.
func (m *restHandler) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	roomName := rq.URL.Path[len("/rest/"):]
	room := m.rooms.FindRoom(roomName)
	if room == nil {
		log.Println("ServeHTTP: room not found ", roomName)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if rq.Method == "GET" {
		m.getMessages(room, w)
	}
	if rq.Method == "POST" {
		m.sendMessages(room,w,rq)
	}
}

//sendMessages handles REST requests for messages and writes them to the response.
func (m *restHandler) sendMessages (room *Room, w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	message := new(restMessage)
	err := dec.Decode(message)
	if err != nil {
		log.Println("Error decoding messages in sendMessages",err)
	}
	message.Time = time.Now()
	room.Send(message)
	m.log(message.String())
}

//log logs REST messages sent to the chatlog.
func (m *restHandler) log (s string) {
	var err error
	_, err = io.WriteString(m.chl, s + "\n")
	if err != nil {
		log.Println(err)
	}
}


//GetMessage handles REST request for messages and writes them to the response.
func (m *restHandler) getMessages(room *Room, w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	messages := room.GetMessages()
	err := enc.Encode(messages)
	if err != nil {
		log.Println("Error encoding messages in restHandler", err)
	}
}

func main() {
	c := configure("Config")
	rooms := NewRoomList()
	chl, err := os.OpenFile(c.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	defer chl.Close()
	if err != nil {
		log.Panic(err)
	}
	tserv := NewTelnetServer(rooms,chl,c)
	go tserv.Start()
	go serverHTTP(rooms,chl,c)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	_ = <-ch
	tserv.Stop()
	chl.Close()
}
