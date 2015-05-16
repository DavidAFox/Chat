package postgres

//A postgresql implementation of datastore for use with clientdata.

import (
	"database/sql"
	"fmt"
	"github.com/davidafox/chat/clientdata"
	_ "github.com/lib/pq"
	"log"
	"strconv"
)

//NewFactory creates a new clientdata factory using a postgres datastore.
func NewFactory(databaseLogin, databasePassword, databaseName, databaseIP, databasePort string) (*Factory, error) {
	cdf := new(Factory)
	database, err := NewPostgres(databaseLogin, databasePassword, databaseName, databaseIP, databasePort)
	cdf.database = database
	return cdf, err
}

//Factory is initialized with the config file and then used to make client data objects of the appropriate type when clients are created.
type Factory struct {
	database clientdata.DataStore
}

//Create returns a ClientData object of the type for the factory.
func (cdf *Factory) Create(name string) clientdata.ClientData {
	return clientdata.NewDataAccess(name, cdf.database)
}

//Postgres is a type of datastore using a postgresql database.
type Postgres struct {
	data *sql.DB
}

//NewPostgres sets up the connection to the database and returns a Postgres datastore.
func NewPostgres(databaseLogin, databasePassword, databaseName, databaseIP, databasePort string) (*Postgres, error) {
	p := new(Postgres)
	if databaseIP == "" {
		databaseIP = "localhost"
	}
	if databasePort == "" {
		databasePort = "5432"
	}
	database, err := sql.Open("postgres", fmt.Sprintf("user=%v password=%v dbname=%v host=%v port=%v sslmode=disable", databaseLogin, databasePassword, databaseName, databaseIP, databasePort))
	p.data = database
	return p, err
}

//Add adds row values to table.
func (p *Postgres) Add(table string, values map[string]string) error {
	qstring := "INSERT INTO " + table + " ("
	args := make([]interface{}, 0, len(values))
	for i := range values {
		qstring += i + ", "
		args = append(args, values[i])
	}
	qstring = qstring[:len(qstring)-2] + ") VALUES ("
	x := 1
	for range values {
		qstring += "$" + strconv.Itoa(x) + ", "
		x++
	}
	qstring = qstring[:len(qstring)-2] + ")"
	_, err := p.data.Exec(qstring, args...)
	return err
}

//Delete removes rows matching values from table.
func (p *Postgres) Delete(table string, values map[string]string) error {
	qstring := "DELETE FROM " + table + " WHERE "
	args := make([]interface{}, 0, len(values)*2)
	x := 1
	for i := range values {
		if x != 1 {
			qstring += " AND "
		}
		qstring += i + " = $" + strconv.Itoa(x)
		args = append(args, values[i])
		x++
	}
	_, err := p.data.Exec(qstring, args...)
	return err
}

//Get gets teh columns from the table for the rows matching values.
func (p *Postgres) Get(table string, values map[string]string, columns ...string) ([]map[string]string, error) {
	qstring := "SELECT "
	args := make([]interface{}, 0)
	if len(columns) > 0 {
		for i := range columns {
			if i != 0 {
				qstring += ", "
			}
			qstring += columns[i]
		}
	} else {
		qstring += "*"
	}
	qstring += " FROM " + table + " WHERE "
	x := 1
	for i := range values {
		if x != 1 {
			qstring += " AND "
		}
		qstring += i + " = $" + strconv.Itoa(x)
		args = append(args, values[i])
		x++
	}
	rows, err := p.data.Query(qstring, args...)
	if err != nil {
		return nil, err
	}

	col := make([]string, 0)
	m := make([]map[string]string, 0)
	y := 0
	for rows.Next() {
		col, err = rows.Columns()
		res := make([]interface{}, len(col))
		st := make([]string, len(col))
		for i := range res {
			res[i] = &st[i]
		}
		rows.Scan(res...)
		if err != nil {
			log.Println("Error in get: ", err)
		}
		m = append(m, make(map[string]string))
		for i := range col {
			m[y][col[i]] = *res[i].(*string)
		}
		y++
	}
	return m, nil
}

//Set finds a row matching cond in table and sets the column/value pairs in values.
func (p *Postgres) Set(table string, values, cond map[string]string) error {
	qstring := "UPDATE " + table + " SET "
	args := make([]interface{}, 0, len(values))
	x := 1
	for i := range values {
		if x != 1 {
			qstring += " AND "
		}
		qstring += i + " = $" + strconv.Itoa(x)
		args = append(args, values[i])
		x++
	}
	qstring += " WHERE "
	args2 := make([]interface{}, 0, len(cond)*2)
	for i := range cond {
		if x != 1+len(values) {
			qstring += " AND "
		}
		qstring += i + " = $" + strconv.Itoa(x+1)
		args2 = append(args2, cond[i])
		x++
	}
	args = append(args, args2...)
	_, err := p.data.Exec(qstring, args)
	return err
}

//Exists takes a table to check and a map representing a row to compare to and returns true if there is a match in the database.
func (p *Postgres) Exists(table string, values map[string]string) (bool, error) {
	qstring := "SELECT count(*) FROM " + table + " WHERE "
	args := make([]interface{}, 0, len(values))
	x := 1
	for i := range values {
		if x != 1 {
			qstring += " AND "
		}
		qstring += i + " = $" + strconv.Itoa(x)
		args = append(args, values[i])
		x++
	}
	var num int
	err := p.data.QueryRow(qstring, args...).Scan(&num)
	if err != nil {
		log.Println(err)
	}
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	case num == 0:
		return false, err
	default:
		return true, nil
	}
}
