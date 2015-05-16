package testclientdata

import (
	"container/list"
	"github.com/davidafox/chat/clientdata"
)

type TestData struct {
	name    string
	blocked *list.List
}

func (td *TestData) Authenticate(pword string) (bool, error) {
	return true, nil
}

func (td *TestData) ClientExists(name string) (bool, error) {
	return false, nil
}

func (td *TestData) NewClient(pword string) error {
	return nil
}

func (td *TestData) IsBlocked(name string) (bool, error) {
	blocked := false
	for i := td.blocked.Front(); i != nil; i = i.Next() {
		if i.Value == name {
			blocked = true
		}
	}
	return blocked, nil
}

func (td *TestData) Block(name string) error {
	if b, _ := td.IsBlocked(name); b {
		return clientdata.ErrBlocking
	}
	td.blocked.PushBack(name)
	return nil
}

func (td *TestData) Unblock(name string) error {
	found := false
	for i, x := td.blocked.Front(), td.blocked.Front(); i != nil; {
		x = i
		i = i.Next()
		if x.Value == name {
			td.blocked.Remove(x)
			found = true
		}
	}
	if found {
		return nil
	} else {
		return clientdata.ErrNotBlocking
	}
}

func (td *TestData) SetName(name string) {
	td.name = name
}

type Factory struct {
}

func (f *Factory) Create(name string) clientdata.ClientData {
	return NewTestData(name)
}

func NewTestData(name string) *TestData {
	td := new(TestData)
	td.name = name
	td.blocked = new(list.List)
	return td
}
