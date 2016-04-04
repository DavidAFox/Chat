package main

/* A Chat Server */
import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/DavidAFox/Chat/client"
	"github.com/DavidAFox/Chat/clientdata"
	"github.com/DavidAFox/Chat/clientdata/datafactory"
	chathttp "github.com/DavidAFox/Chat/connections/http"
	"github.com/DavidAFox/Chat/connections/telnet"
	"github.com/DavidAFox/Chat/message"
	"github.com/DavidAFox/Chat/room"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

//config stores the configuration data from the config file.
type config struct {
	ListeningIP          string
	ListeningPort        string
	HTTPListeningIP      string
	HTTPListeningPort    string
	TLSListeningIP       string
	TLSListeningPort     string
	TLSHTTPListeningIP   string
	TLSHTTPListeningPort string
	CertFile             string
	KeyFile              string
	LogFile              string
	DatabaseIP           string
	DatabasePort         string
	DatabaseLogin        string
	DatabasePassword     string
	DatabaseName         string
	DatabaseType         string
	Origin               string
	MaxRooms             int
	DisableNewAccounts   bool
}

//configure loads the config file.
func configure(filename string) (c *config) {
	file, err := os.Open(filename)
	if err != nil {
		log.Panic("Error opening config file", err)
	}
	dec := json.NewDecoder(file)
	c = new(config)
	err = dec.Decode(c)
	if err != nil {
		log.Panic("Error decoding config file", err)
	}
	return
}

type telnetServer struct {
	rooms       *room.RoomList
	chatlog     io.WriteCloser
	cls         chan bool
	ln          net.Listener
	done        bool
	datafactory clientdata.Factory
}

//NewTelnetServerTLS creates a telnet server using TLS.
func NewTelnetServerTLS(rooms *room.RoomList, chl io.WriteCloser, c *config, datafactory clientdata.Factory) *telnetServer {
	ts := new(telnetServer)
	var err error
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		log.Panic(err)
	}
	conf := new(tls.Config)
	conf.Certificates = append(conf.Certificates, cert)
	ts.ln, err = tls.Listen("tcp", net.JoinHostPort(c.TLSListeningIP, c.TLSListeningPort), conf)
	if err != nil {
		log.Panic(err)
	}
	ts.cls = make(chan bool, 1)
	ts.rooms = rooms
	ts.chatlog = chl
	ts.done = false
	ts.datafactory = datafactory
	return ts
}

func NewTelnetServer(rooms *room.RoomList, chl io.WriteCloser, c *config, datafactory clientdata.Factory) *telnetServer {
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
	ts.datafactory = datafactory
	return ts
}

func (ts *telnetServer) Stop() {
	ts.done = true
	ts.cls <- true
	ts.ln.Close()
}

//server listens for connections and sends them to handleConnection().
func (ts *telnetServer) Start() {
Outerloop:
	for {
		select {
		case <-ts.cls:
			break Outerloop
		default:
			conn, err := ts.ln.Accept()
			if err != nil && ts.done == false {
				log.Println(err)
			}
			if conn != nil {
				go telnet.TelnetLogin(conn, ts.rooms, ts.chatlog, ts.datafactory.Create(""))
			}
		}
	}
}

//serverHTTPTLS sets up the http handlers and then runs ListenAndServeTLS.
func serverHTTPTLS(rooms *room.RoomList, chl io.WriteCloser, c *config, df clientdata.Factory) {
	mux := http.NewServeMux()
	room := chathttp.NewRoomHandler(chathttp.Options{RoomList: rooms, ChatLog: chl, DataFactory: df, ClientFactory: client.NewFactory(rooms, chl, df), Origin: c.Origin})
	mux.Handle("/", room)
	rest := newRestHandler(rooms, chl)
	mux.Handle("/rest/", rest)
	err := http.ListenAndServeTLS(net.JoinHostPort(c.TLSHTTPListeningIP, c.TLSHTTPListeningPort), c.CertFile, c.KeyFile, mux)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

//serverHTTP sets up the http handlers and then runs ListenAndServe
func serverHTTP(rooms *room.RoomList, chl io.WriteCloser, c *config, df clientdata.Factory) {
	mux := http.NewServeMux()
	room := chathttp.NewRoomHandler(chathttp.Options{RoomList: rooms, ChatLog: chl, DataFactory: df, ClientFactory: client.NewFactory(rooms, chl, df), Origin: c.Origin})
	mux.Handle("/", room)
	rest := newRestHandler(rooms, chl)
	mux.Handle("/rest/", rest)
	err := http.ListenAndServe(net.JoinHostPort(c.HTTPListeningIP, c.HTTPListeningPort), mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

//restHandler is the http.Handler for handling the REST API
type restHandler struct {
	rooms *room.RoomList
	chl   io.WriteCloser
}

//newRestHandler initializes a new restHandler.
func newRestHandler(rooms *room.RoomList, chl io.WriteCloser) *restHandler {
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
		m.sendMessages(room, w, rq)
	}
}

//sendMessages handles REST requests for messages and writes them to the response.
func (m *restHandler) sendMessages(room *room.Room, w http.ResponseWriter, rq *http.Request) {
	dec := json.NewDecoder(rq.Body)
	message := new(message.RestMessage)
	err := dec.Decode(message)
	if err != nil {
		log.Println("Error decoding messages in sendMessages", err)
	}
	message.Time = time.Now()
	room.Send(message)
	m.log(message.String())
}

//log logs REST messages sent to the chatlog.
func (m *restHandler) log(s string) {
	var err error
	_, err = io.WriteString(m.chl, s+"\n")
	if err != nil {
		log.Println(err)
	}
}

//GetMessage handles REST request for messages and writes them to the response.
func (m *restHandler) getMessages(room *room.Room, w http.ResponseWriter) {
	enc := json.NewEncoder(w)
	messages := room.GetMessages()
	err := enc.Encode(messages)
	if err != nil {
		log.Println("Error encoding messages in restHandler", err)
	}
}

type NoLog struct {
}

func (nl NoLog) Write(b []byte) (int, error) {
	return len(b), nil
}

func (nl NoLog) Close() error {
	return nil
}

func main() {
	loc := flag.String("config", "Config", "the location of the config file")
	flag.Parse()
	c := configure(*loc)
	rooms := room.NewRoomList(c.MaxRooms)
	defer rooms.Close()
	var chl io.WriteCloser
	var err error
	if c.LogFile != "" {
		chl, err = os.OpenFile(c.LogFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
		defer chl.Close()
		if err != nil {
			log.Panic(err)
		}
	} else {
		chl = new(NoLog)
	}
	df, err := datafactory.New(c.DatabaseType, c.DatabaseLogin, c.DatabasePassword, c.DatabaseName, c.DatabaseIP, c.DatabasePort, c.DisableNewAccounts)
	if err != nil {
		log.Panic(err)
	}
	if c.ListeningPort != "" {
		tserv := NewTelnetServer(rooms, chl, c, df)
		fmt.Println("Starting Telnet Server on Port ", c.ListeningPort)
		go tserv.Start()
		defer tserv.Stop()
	}
	if c.TLSListeningPort != "" {
		tlstserv := NewTelnetServerTLS(rooms, chl, c, df)
		fmt.Println("Starting TLS Telnet Server on Port ", c.TLSListeningPort)
		go tlstserv.Start()
		defer tlstserv.Stop()
	}
	if c.TLSHTTPListeningPort != "" {
		fmt.Println("Starting TLS HTTP Server on Port ", c.TLSHTTPListeningPort)
		go serverHTTPTLS(rooms, chl, c, df)
	}
	if c.HTTPListeningPort != "" {
		fmt.Println("Starting HTTP Server on Port ", c.HTTPListeningPort)
		go serverHTTP(rooms, chl, c, df)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	_ = <-ch
}
