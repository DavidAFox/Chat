package clientdata

//clientdata is a package for handling persistent data storage related to clients for the chat server.  Different forms of data storage can be used by using the DataStore interface.

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"regexp"
	"sort"
	"time"
)

type Factory interface {
	Create(name string) ClientData
}

//ClientData is the interface used for interacting with the persistent client data.
type ClientData interface {
	Authenticate(pword string) (bool, error)
	ClientExists(name string) (bool, error)
	LastOnline(name string) (time.Time, error)
	UpdateOnline(t time.Time) error
	NewClient(pword string) error
	IsBlocked(name string) (bool, error)
	BlockList() ([]string, error)
	Block(name string) error
	Unblock(name string) error
	IsFriend(name string) (bool, error)
	Friend(name string) error
	Unfriend(name string) error
	FriendList() ([]string, error)
	SetName(name string)
}

//DataStore is the interface used by DataAccess to access stored data.
type DataStore interface {
	Add(table string, values map[string]string) error
	Delete(table string, values map[string]string) error
	Get(table string, values map[string]string, columns ...string) ([]map[string]string, error)
	Set(table string, values, cond map[string]string) error
	Exists(table string, values map[string]string) (bool, error)
}

var ErrClientExists = errors.New("clientdata: Client already exists")
var ErrNotBlocking = errors.New("clientdata: You are not blocking them.")
var ErrClientNotFound = errors.New("clientdata: Client not found.")
var ErrInvalidName = errors.New("clientdata: Invalid Name.")
var ErrBlocking = errors.New("clientdata: You are already blocking them.")
var ErrFriend = errors.New("clientdata: They are already on your friends list.")
var ErrNotFriend = errors.New("clientdata: They are not on your friends list.")
var ErrAccountCreationDisabled = errors.New("clientdata: New account creation has been disabled.")

//encrypt encrypts the password and returns the encrypted version.
func Encrypt(pword string) string {
	crypass, err := bcrypt.GenerateFromPassword([]byte(pword), 12)
	if err != nil {
		log.Println("Error encrypting pass: ", err)
	}
	return string(crypass)
}

//ValidateName returns true if a name is only alphanumeric characters.
func ValidateName(name string) bool {
	inv, err := regexp.MatchString("[^[:alnum:]]", name)
	if err != nil {
		log.Println("Error in ValidateName: ", err)
	}
	return !inv
}

//DataAccess is the default type of ClientData.
type DataAccess struct {
	name               string
	data               DataStore
	disableNewAccounts bool
}

//NewDataAccess creates a new DataAccess.  Names must be alphanumeric only.
func NewDataAccess(name string, data DataStore, disableNewAccounts bool) *DataAccess {
	cdd := new(DataAccess)
	if ValidateName(name) {
		cdd.name = name
	}
	cdd.data = data
	cdd.disableNewAccounts = disableNewAccounts
	return cdd
}

//Authenticate returns true if the password matches the clients password.
func (cdd *DataAccess) Authenticate(pword string) (bool, error) {
	res, err := cdd.data.Get("client", row("name", cdd.name), "password")
	switch {
	case err == ErrClientNotFound:
		return false, nil
	case err != nil:
		return false, err
	case len(res) > 1:
		return false, errors.New("Client has more than one password stored")
	case len(res) == 0:
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(res[0]["password"]), []byte(pword))
	switch {
	case err == nil:
		return true, nil
	case err == bcrypt.ErrMismatchedHashAndPassword:
		return false, nil
	default:
		return false, err
	}
}

//ClientExists returns true if a client with name is in the database
func (cdd *DataAccess) ClientExists(name string) (bool, error) {
	res, err := cdd.data.Exists("client", row("name", name))
	switch {
	case err == ErrClientNotFound:
		return false, nil
	case err != nil:
		return false, err
	default:
		return res, nil
	}
}

//UpdateOnline sets the clients lastonline to time.
func (cdd *DataAccess) UpdateOnline(t time.Time) error {
	return cdd.data.Set("client", row("lastonline", t.String()), row("name", cdd.name))
}

//LastOnline returns the clients lastonline entry.
func (cdd *DataAccess) LastOnline(name string) (time.Time, error) {
	res, err := cdd.data.Get("client", row("name", name), "lastonline")
	if err != nil {
		return time.Now(), err
	}
	if len(res) == 0 {
		return time.Now(), ErrClientNotFound
	}
	t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", res[0]["lastonline"])
	return t, err
}

//NewClient adds the client to the database with the provided password.
func (cdd *DataAccess) NewClient(pword string) error {
	if cdd.disableNewAccounts {
		return ErrAccountCreationDisabled
	}
	exists, err := cdd.ClientExists(cdd.name)
	switch {
	case err != nil:
		return err
	case exists:
		return ErrClientExists
	}
	if !ValidateName(cdd.name) {
		return ErrInvalidName
	}
	hashpword := Encrypt(pword)
	err = cdd.data.Add("client", row("password", hashpword, "name", cdd.name, "lastonline", time.Now().String()))
	return err
}

//BlockList returns a list of names that the client is blocking.
func (cdd *DataAccess) BlockList() ([]string, error) {
	rows, err := cdd.data.Get("blocked", row("name", cdd.name), "blocked")
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, 0)
	for _, i := range rows {
		list = append(list, i["blocked"])
	}
	sort.Strings(list)
	return list, nil
}

//IsBlocked returns true if the client is blocking name.
func (cdd *DataAccess) IsBlocked(name string) (bool, error) {
	return cdd.data.Exists("blocked", row("blocked", name, "name", cdd.name))
}

//Block adds name to the clients blocklist.
func (cdd *DataAccess) Block(name string) error {
	if !ValidateName(name) {
		return ErrInvalidName
	}
	blocked, err := cdd.IsBlocked(name)
	if err != nil {
		return err
	}
	if blocked {
		return ErrBlocking
	}
	return cdd.data.Add("blocked", row("blocked", name, "name", cdd.name))
}

//Unblock removes name from the client's blocklist.  It will return ErrNotBlocking if name isn't on the client's blocklist.
func (cdd *DataAccess) Unblock(name string) error {
	if !ValidateName(name) {
		return ErrInvalidName
	}
	blocked, err := cdd.IsBlocked(name)
	if err != nil {
		return err
	}
	if !blocked {
		return ErrNotBlocking
	}
	return cdd.data.Delete("blocked", row("blocked", name, "name", cdd.name))
}

//IsFriend returns true if name is in the friendlist
func (cdd *DataAccess) IsFriend(name string) (bool, error) {
	return cdd.data.Exists("friends", row("friend", name, "name", cdd.name))
}

//Friend adds name to the friend list.
func (cdd *DataAccess) Friend(name string) error {
	if !ValidateName(name) {
		return ErrInvalidName
	}
	friend, err := cdd.IsFriend(name)
	if err != nil {
		return err
	}
	if friend {
		return ErrFriend
	}
	return cdd.data.Add("friends", row("friend", name, "name", cdd.name))
}

//FriendList returns a [] of the names on the clients friend list.
func (cdd *DataAccess) FriendList() ([]string, error) {
	rows, err := cdd.data.Get("friends", row("name", cdd.name), "friend")
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, 0)
	for i := range rows {
		list = append(list, rows[i]["friend"])
	}
	sort.Strings(list)
	return list, nil
}

//Unfriend removes name from the friend list.
func (cdd *DataAccess) Unfriend(name string) error {
	if !ValidateName(name) {
		return ErrInvalidName
	}
	friend, err := cdd.IsFriend(name)
	if err != nil {
		return err
	}
	if !friend {
		return ErrNotFriend
	}
	return cdd.data.Delete("friends", row("friend", name, "name", cdd.name))
}

//SetName changes the name associated with this DataAccess object.  Name must be alphanumeric only.
func (cdd *DataAccess) SetName(name string) {
	if ValidateName(name) {
		cdd.name = name
	}
}

//row is a helper function for creating a map from a set of column/value pairs.
func row(x ...string) map[string]string {
	if len(x)%2 != 0 {
		log.Println("Error in clientdata.row() must be even number of arguments.")
		return nil
	}
	m := make(map[string]string)
	for i := 0; i < len(x); i += 2 {
		m[x[i]] = x[i+1]
	}
	return m
}
