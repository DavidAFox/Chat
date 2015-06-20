package room

import ()

//RoomList is a linked list of rooms with a mutex.
type RoomList struct {
	*clientList
}

//NewRoomList returns an empty RoomList.
func NewRoomList() *RoomList {
	rl := &RoomList{NewClientList()}
	rl.Add(NewRoom("Lobby"))//create default room
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
