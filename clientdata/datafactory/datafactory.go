package datafactory

import (
	//Add new data packages here.
	"github.com/davidafox/chat/clientdata"
	"github.com/davidafox/chat/clientdata/filedata"
	"github.com/davidafox/chat/clientdata/postgres"
)

//NewFactory returns a factory to make client data objects of the type kind and using the database.  Currently supports "postgres" as a kind, using a Postgres database.
func New(kind, databaseLogin, databasePassword, databaseName, databaseIP, databasePort string) (clientdata.Factory, error) {
	if kind == "postgres" {
		return postgres.NewFactory(databaseLogin, databasePassword, databaseName, databaseIP, databasePort)
	}
	return filedata.NewFactory(databaseName), nil
}
