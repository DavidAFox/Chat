package filedata

import (
	"encoding/json"
	"errors"
	"github.com/davidafox/chat/clientdata"
	"log"
	"os"
	"sync"
)

//DEFAULTFILENAME is the name the data object will use for storing the data if one is not provided.
var DEFAULTFILENAME = "ClientDataFile"

//Factory is a factory for creating ClientData objects using fileData.
type Factory struct {
	data *fileData
}

//NewFactory returns a Factory that will make client data objects using the filedata as its source.
func NewFactory(fileName string) *Factory {
	f := new(Factory)
	f.data = NewFileData(fileName)
	return f
}

//Create makes a new ClientData object using the factories filedata as its source.
func (cdf *Factory) Create(name string) clientdata.ClientData {
	cd := clientdata.NewDataAccess(name, cdf.data)
	return cd
}

//fileData is a data sourc for the ClientData object using a set of maps and storing them in a file.
type fileData struct {
	Records map[string]*ClientRecord
	*sync.RWMutex
	FileName string
	get      chan string
	send     chan *ClientRecord
}

//NewFileData creates a new file data object loading the existing file or making a new one if one does not exist.
func NewFileData(fileName string) *fileData {
	fd := new(fileData)
	fd.Records = make(map[string]*ClientRecord)
	fd.FileName = fileName
	fd.get = make(chan string)
	fd.send = make(chan *ClientRecord)
	go fd.recordMaker()
	if fileName == "" {
		fileName = DEFAULTFILENAME
	}
	file, err := os.Open(fileName)
	defer file.Close()
	switch {
	case os.IsNotExist(err):
		log.Printf("No client data file found.  A new one will be created.")
	case err != nil:
		log.Printf("Error opening data file %v. %v\n", fileName, err)
	default:
		dec := json.NewDecoder(file)
		err = dec.Decode(&fd.Records)
		if err != nil {
			log.Println("Error decoding data file in NewFileData: ", err)
		}
	}
	for _, x := range fd.Records {
		x.RWMutex = new(sync.RWMutex)
	}
	fd.RWMutex = new(sync.RWMutex)
	return fd
}

//Add adds a row continging the key value pairs from values.
func (fd *fileData) Add(table string, values map[string]string) error {
	r, found := fd.Records[values["name"]]
	if !found {
		r = fd.NewRecord(values["name"])
	}
	fd.RLock()
	r.Lock()
	r.Tables[table] = append(r.Tables[table], values)
	r.Unlock()
	fd.RUnlock()
	return fd.save()
}

//Delete will delete all rows from table that match all of the key value pairs in values
func (fd *fileData) Delete(table string, values map[string]string) error {
	r, found := fd.Records[values["name"]]
	if !found {
		return clientdata.ErrClientNotFound
	}
	fd.RLock()
	r.Lock()
	for i := range r.Tables[table] {
		if matchRow(r.Tables[table][i], values) {
			r.Tables[table] = append(r.Tables[table][:i], r.Tables[table][i+1:]...)
			i--
		}
	}
	r.Unlock()
	fd.RUnlock()
	return fd.save()
}

//copyMap returns a copy of the original map.
func copyMap(original map[string]string) map[string]string {
	n := make(map[string]string)
	for i, j := range original {
		n[i] = j
	}
	return n
}

//Get returns slice of maps representing the tables that match the row represented by values.  If columns are provided it will return only those columns in the rows returned.
func (fd *fileData) Get(table string, values map[string]string, columns ...string) ([]map[string]string, error) {
	r, found := fd.Records[values["name"]]
	res := make([]map[string]string, 0)
	if !found {
		return nil, clientdata.ErrClientNotFound
	}
	r.RLock()
	for i := range r.Tables[table] {
		if matchRow(r.Tables[table][i], values) {
			if len(columns) == 0 {
				res = append(res, copyMap(r.Tables[table][i]))
			} else {
				nmap := make(map[string]string)
				for x := range columns {
					nmap[columns[x]] = r.Tables[table][i][columns[x]]
				}
				res = append(res, nmap)
			}
		}
	}
	r.RUnlock()
	return res, nil
}

//Set sets the values of the rows matching cond to those in values.
func (fd *fileData) Set(table string, values, cond map[string]string) error {
	r, found := fd.Records[cond["name"]]
	if _, ok := values["name"]; ok {
		return errors.New("Err cannot set name with filedata")
	}
	if !found {
		return clientdata.ErrClientNotFound
	}
	fd.RLock()
	r.Lock()
	for i := range r.Tables[table] {
		if matchRow(r.Tables[table][i], cond) {
			for x := range values {
				r.Tables[table][i][x] = values[x]
			}
		}
	}
	r.Unlock()
	fd.RUnlock()
	return fd.save()
}

//Exists will return true if a row matching all the pairs in values exists.
func (fd *fileData) Exists(table string, values map[string]string) (bool, error) {
	if _, ok := fd.Records[values["name"]]; !ok {
		return false, nil
	}
	if _, ok := fd.Records[values["name"]].Tables[table]; !ok {
		return false, nil
	}
	fd.Records[values["name"]].Lock()
	defer fd.Records[values["name"]].Unlock()
	for i := range fd.Records[values["name"]].Tables[table] {
		if matchRow(fd.Records[values["name"]].Tables[table][i], values) {
			return true, nil
		}
	}
	return false, nil
}

//matchRow returns true if row contains all the fileds of matchvalue and they match otherwise it returns false.
func matchRow(row, matchvalue map[string]string) bool {
	y := true
	for x := range matchvalue {
		if d, ok := row[x]; !ok || d != matchvalue[x] {
			y = false
		}
	}
	return y
}

//save writes the map to the file.
func (fd *fileData) save() error {
	fd.Lock()
	defer fd.Unlock()
	tmp, err := os.Create(fd.FileName + ".tmp")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(tmp)
	err = enc.Encode(fd.Records)
	if err != nil {
		return err
	}
	err = tmp.Close()
	if err != nil {
		return err
	}
	err = os.Remove(fd.FileName)
	if err != nil && !os.IsNotExist(err) {
		log.Println(err)
	}
	err = os.Rename(fd.FileName+".tmp", fd.FileName)
	if err != nil {
		return err
	}
	return nil
}

//ClientRecord represents all the data associated with a single client.
type ClientRecord struct {
	*sync.RWMutex
	//    (table name)(row number)(column name)
	Tables map[string][]map[string]string
}

//NewRecord returns a new client record.
func (fd *fileData) NewRecord(name string) *ClientRecord {
	fd.get <- name
	return <-fd.send
}

//recordMaker is a function that controls the creation of client records in the main map.
func (fd *fileData) recordMaker() {
	for {
		name := <-fd.get
		if _, found := fd.Records[name]; !found {
			fd.Records[name] = newClientRecord()
		}
		fd.send <- fd.Records[name]
	}
}

//NewClientRecord is used by the record maker to make new client records.
func newClientRecord() *ClientRecord {
	r := new(ClientRecord)
	r.Tables = make(map[string][]map[string]string)
	r.RWMutex = new(sync.RWMutex)
	return r
}
