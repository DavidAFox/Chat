package connections

import (
	"github.com/davidafox/chat/message"
)

type Client interface {
	Execute(command []string) Response
	LeaveRoom()
	SetConnection(Connection)
}

type Response interface {
	Success() bool
	Code() int
	String() string
	Data() interface{}
}

type ClientFactory interface {
	New(name string, connection Connection) Client
}

type Connection interface {
	SendMessage(m message.Message)
	Close()
}
