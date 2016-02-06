package datafactory

import (
	//Add new data packages here.
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/clientdata/filedata"
	"github.com/davidafox/chat/clientdata/postgres"
)

//NewFactory returns a factory to make client data objects of the type kind and using the database.  Currently supports "postgres" as a kind, using a Postgres database.
func New(kind, databaseLogin, databasePassword, databaseName, databaseIP, databasePort string, disableNewAccounts bool) (clientdata.Factory, error) {
	if kind == "postgres" {
		data, err := postgres.NewPostgres(databaseLogin, databasePassword, databaseName, databaseIP, databasePort)
		return NewDataFactory(data, disableNewAccounts), err
	}
	return NewDataFactory(filedata.NewFileData(databaseName), disableNewAccounts), nil
}

type DataFactory struct {
	data               clientdata.DataStore
	disableNewAccounts bool
}

func (df *DataFactory) Create(name string) clientdata.ClientData {
	ca := clientdata.NewDataAccess(name, df.data, df.disableNewAccounts)
	return ca
}

func NewDataFactory(data clientdata.DataStore, disableNewAccounts bool) *DataFactory {
	df := new(DataFactory)
	df.data = data
	df.disableNewAccounts = disableNewAccounts
	return df
}
