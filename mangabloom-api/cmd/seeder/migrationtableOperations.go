package main

import (
	"database/sql"
	"log"
	"time"
)

//after recording migration begin and end times,
//we have to scan the values back into the migration table struct
//so that we can use the id for updating the end time after migration is done

func migrationStart(migrationTable *MigrationTable) {
	rows, err := dbConnection.Query("INSERT into migration_table(migration_begin) values($1) RETURNING *", time.Now().UTC())
	if err != nil {
		log.Fatal("failed to insert migration begin time: ", err)
	}

	defer rows.Close()
	handleRowScanning(rows, migrationTable)
}

func migrationEnd(migrationTable *MigrationTable) {
	rows, err := dbConnection.Query("UPDATE migration_table SET migration_end = $1 WHERE id = $2 RETURNING *;", time.Now().UTC(), migrationTable.ID)
	if err != nil {
		log.Fatal("failed to update migration end time: ", err)
	}

	defer rows.Close()
	handleRowScanning(rows, migrationTable)
}

func handleRowScanning(rows *sql.Rows, migration_table *MigrationTable) {
	for rows.Next() {
		err := rows.Scan(&migration_table.ID, &migration_table.MigrationBegin, &migration_table.MigrationEnd)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
