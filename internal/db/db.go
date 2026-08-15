package db

import(
	"database/sql"
	"fmt"
	"github.com/krishnabhardwaj25/flowithgo/internal/logger"
	_"github.com/lib/pq"

)

func Connect(databaseUrl string)(*sql.DB,error){
	db,err := sql.Open("postgres",databaseUrl)
	if err != nil{
		return nil, fmt.Errorf("failed to open the db %w",err)
	}
	if err = db.Ping() ; err != nil{
		return nil , fmt.Errorf("failed to ping the db %w",err)
	}
	logger.L.Info("connected")
	return db,nil

}