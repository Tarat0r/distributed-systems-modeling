package metrics

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	_ "github.com/mattn/go-sqlite3"
)

const PathDB = "metrics/merics.db"

var ExperimentStartTime string

func AgregateToDB(tableName string) {
	// This function will aggregate metrics from CSV files into a database.

	db, err := sql.Open("sqlite3", PathDB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable(db, tableName)

	dir := "metrics"
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	writeStartTimeToDB(db, tableName, ExperimentStartTime)

	// Iterate through all files csv in the metrics directory
	for _, CSVfile := range files {
		if !CSVfile.IsDir() && strings.HasSuffix(CSVfile.Name(), ".csv") {

			// Open CSV file
			file, err := os.Open(filepath.Join(dir, CSVfile.Name()))
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil {
				log.Fatal(err)
			}

			writeCSVtoDB(db, tableName, records)
			flags.VPrintf("Data from %s aggregated to database successfully\n", CSVfile.Name())

			// Remove the CSV file after processing
			removeCSVFile(dir, CSVfile)
		}
	}
}

func createTable(db *sql.DB, tableName string) {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS ` + tableName + ` (
	TransactionID INTEGER PRIMARY KEY AUTOINCREMENT,
	ExperimentID  INTEGER,
	Time          DATETIME,
	NodeID        INTEGER,
	AliveNode     BOOLEAN,
	SenderID      INTEGER,
	ReceiverID    INTEGER,
	MessagesData  TEXT,
	MessageID     INTEGER,
	NodeDB        TEXT,
	MessageType   TEXT
	);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	flags.VPrintln("Table", tableName, "created successfully")

}

func writeCSVtoDB(db *sql.DB, tableName string, records [][]string) {
	for i, row := range records {
		if i == 0 {
			continue // Skip header row
		}
		_, err := db.Exec(`INSERT INTO `+tableName+` (
			ExperimentID, Time, NodeID, AliveNode, SenderID, ReceiverID, MessagesData, MessageID, NodeDB, MessageType) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)`, row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9])
		if err != nil {
			log.Printf("Insert failed at row %d: %v", i, err)
		}
	}
}

func removeCSVFile(dir string, file os.DirEntry) {
	if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
		err := os.Remove(filepath.Join(dir, file.Name()))
		if err != nil {
			log.Printf("Error removing file %s: %v", file.Name(), err)
		} else {
			flags.VPrintf("File %s removed successfully\n", file.Name())
		}
	}
}

func RemoveDB() error {
	_, err := os.Stat(PathDB)
	if err != nil {
		flags.VPrintln("Database file", PathDB, "does not exist, nothing to remove")
		return nil
	}

	db, err := sql.Open("sqlite3", PathDB)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}
	defer db.Close()

	// Optional: wait instead of failing instantly if locked
	db.Exec("PRAGMA busy_timeout = 5000")

	// Read table names
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table'`)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		if table != "sqlite_sequence" {
			tables = append(tables, table)
		}
	}
	rows.Close() // important: VACUUM requires no active query

	// Clear each table
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM `%s`", table))
		if err != nil {
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
		fmt.Println("Cleared table:", table)
	}

	// Optionally reset autoincrement counters
	_, _ = db.Exec("DELETE FROM sqlite_sequence")

	// Run VACUUM to clean up file space
	_, err = db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("VACUUM failed: %w", err)
	}

	return nil
}

func AddExperimentStartTime() {
	ExperimentStartTime = time.Now().Format("2006-01-02 15:04:05.000000000")
	flags.VPrintln("Added start time for experiment:", ExperimentStartTime)
}

func writeStartTimeToDB(db *sql.DB, tableName string, ExperimentStartTime string) {
	_, err := db.Exec(`INSERT INTO `+tableName+` (ExperimentID, Time, MessagesData) VALUES (?, ?, ?)`, flags.Exper.ID, ExperimentStartTime, "START")
	if err != nil {
		log.Printf("Failed to write start time to database: %v", err)
	} else {
		flags.VPrintln("Experiment start time written to database successfully")
	}

}

func WriteExperimentToDB() {
	tableName := "Experiments"
	db, err := sql.Open("sqlite3", PathDB)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS ` + tableName + ` (
    ID                    INTEGER PRIMARY KEY,
    Timer                 INTEGER,
    NodeCount             INTEGER,
    MulticastDomains      INTEGER,
    GossipFanOut          INTEGER,
    DelayMean             INTEGER,
    AliveProbability      REAL,
    LossProbability       REAL,
    CorruptionProbability REAL
);`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	flags.VPrintln("Table", tableName, "created successfully")

	_, err = db.Exec(`INSERT INTO Experiments (
		ID,
		Timer,
		NodeCount,
		MulticastDomains,
		GossipFanOut,
		DelayMean,
		AliveProbability,
		LossProbability,
		CorruptionProbability
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		flags.Exper.ID,
		flags.Exper.Timer,
		flags.Exper.NodeCount,
		flags.Exper.MulticastDomains,
		flags.Exper.GossipFanOut,
		flags.Exper.DelayMean,
		flags.Exper.AliveProbability,
		flags.Exper.LossProbability,
		flags.Exper.CorruptionProbability,
	)
	if err != nil {
		log.Printf("Failed to write experiment to database: %v", err)
	} else {
		flags.VPrintln("Experiment written to database successfully")
	}
}
