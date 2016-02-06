package room

import (
	"errors"
	"log"
	"time"
)

var ERR_MAX_ROOMS = errors.New("Can't create room.  There are already the maximum number of rooms.")
var ERR_ROOM_EXISTS = errors.New("A room with that name already exits.")

//RoomList is a linked list of rooms with a mutex.
type RoomList struct {
	maxRooms int
	*clientList
	closeChannel chan bool
}

//NewRoomList returns an empty RoomList.
func NewRoomList(maxRooms int) *RoomList {
	if maxRooms < 1 {
		maxRooms = 1
	}
	rl := &RoomList{maxRooms, NewClientList(), make(chan bool, 1)}
	err := rl.Add(NewRoom("Lobby")) //create default room
	if err != nil {
		log.Println(err)
	}
	go rl.roomManager()
	return rl
}

//FindRoom returns the first room with name.
func (rml *RoomList) FindRoom(name string) *Room {
	for i := rml.Front(); i != nil; i = i.Next() {
		if i.Value.(*Room).Name() == name {
			return i.Value.(*Room)
		}
	}
	return nil
}

//FindClientRoom returns the name of a room that a client with name is in.
func (rml *RoomList) FindClientRoom(name string) string {
	for i := rml.Front(); i != nil; i = i.Next() {
		if i.Value.(*Room).Present(name) {
			return i.Value.(*Room).Name()
		}
	}
	return ""
}

//GetClient returns the first client that matches name.
func (rml *RoomList) GetClient(name string) Client {
	for i := rml.Front(); i != nil; i = i.Next() {
		cl := i.Value.(*Room).GetClient(name)
		if cl != nil {
			return cl
		}
	}
	return nil
}

//CloseEmpty closes all empty rooms.
func (rml *RoomList) CloseEmpty() {
	rml.Lock()
	defer rml.Unlock()
	for entry, x := rml.Front(), rml.Front(); entry != nil; { //Close any empty rooms
		x = entry
		entry = entry.Next()
		if x.Value.(*Room).IsEmpty() {
			rml.Remove(x)
		}
	}
}

func (rml *RoomList) Add(cl Client) error {
	if rml.clientList.count >= rml.maxRooms {
		return ERR_MAX_ROOMS
	}
	if rml.clientList.Present(cl.Name()) {
		return ERR_ROOM_EXISTS
	}
	rml.clientList.Add(cl)
	return nil
}

func (rml *RoomList) roomManager() {
	for {
		for i := rml.clientList.Front(); i != nil; {
			if rm, ok := i.Value.(*Room); ok {
				if rm.clients.count < 1 && rm.Name() != "Lobby" {
					x := i
					i = i.Next()
					rml.clientList.Remove(x)
					rml.clientList.count--
				} else {
					i = i.Next()
				}
			} else {
				log.Println("non-room in roomlist: ", i.Value.(Client).Name())
			}
		}
		select {
		case <-rml.closeChannel:
			return
		default:
			time.Sleep(1 * time.Minute)
		}
	}
}

func (rml *RoomList) Close() {
	rml.closeChannel <- true
}
