package main

import (
)


//RoomList is a linked list of rooms with a mutex.
type RoomList struct {
	*clientList
}

//NewRoomList returns an empty RoomList.
func NewRoomList () *RoomList {
	return &RoomList{NewClientList()}
}

//FindRoom returns the first room with name.
func (rml *RoomList) FindRoom (name string) *Room {
	for i := rml.Front(); i !=nil; i = i.Next() {
		if i.Value.(*Room).Name() == name {
			return i.Value.(*Room)
		}
	}
	return nil
}

//CloseEmpty closes all empty rooms.
func (rml *RoomList) CloseEmpty () {
	rml.Lock()
	defer rml.Unlock()
	for entry,x := rml.Front(), rml.Front(); entry != nil; {//Close any empty rooms
		x = entry
		entry = entry.Next()
		if x.Value.(*Room).Clients.Front() == nil {
			rml.Remove(x)
		}
	}
}
