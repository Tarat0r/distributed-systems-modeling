package analyze

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
)

const ErrNoMessagesFmt = "No messages found for experiment %d in algorithm %s"

var tableName = "AnalyzeResults"

var Summary AnalyzeResults

type AnalyzeResults struct {
	ExperimentID        int
	Algorithm           string
	Time                time.Duration
	Rounds              int
	TotalMessages       int
	MaxSentFromNode     int
	MaxReceivedByNode   int
	OKPercentage        float64
	CorruptedPercentage float64
	LostPercentage      float64
	OKCount             int
	CorruptedCount      int
	LostCount           int
	AliveNodesCount     int
	DeadNodesCount      int
	TimerExpired        bool
}

func Analyze(nodes []*node.Node, algo string) {
	// This function will analyze the results of the simulation.
	// It will read the metrics from the database and print them to the console.
	// You can also implement more complex analysis here, such as statistical analysis or visualization.

	// For now, we will just print a message indicating that the analysis is complete.
	fmt.Println("Analysis complete. Results are available in the database.")

	db, err := sql.Open("sqlite3", metrics.PathDB)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	createTable(db)

	createAnalyzeResults(nodes, db, algo)

	writeToDB(db)
}

func createAnalyzeResults(nodes []*node.Node, db *sql.DB, algo string) {

	var err error
	Summary.ExperimentID = flags.Exper.ID
	Summary.Algorithm = algo
	Summary.Time, err = getTimeDuration(db, flags.Exper.ID, algo)
	if err != nil {
		log.Printf("Error getting time duration: %v", err)
		return
	}
	Summary.Rounds = getRoundsCount(algo)
	Summary.TotalMessages = getTotalMessages(db, flags.Exper.ID, algo)
	Summary.MaxSentFromNode = getMaxSentFromNode(db, flags.Exper.ID, algo)
	Summary.MaxReceivedByNode = getMaxReceivedByNode(db, flags.Exper.ID, algo)
	getDB(nodes) //Get OK, Corrupted and Lost count from nodes
	getAliveAndDeadNodesCount(nodes)

	getPercentages()

}

func createTable(db *sql.DB) {

	sqlStmt := `
	CREATE TABLE IF NOT EXISTS ` + tableName + ` (
	ID					       INTEGER PRIMARY KEY AUTOINCREMENT,
    ExperimentID               INTEGER,
	Algorithm                  TEXT,
    TimeDuration               REAL,
    Rounds                     INTEGER,
    TotalMessages              INTEGER,
    MaxSentFromNode            INTEGER,
    MaxReceivedByNode          INTEGER,
    OKPercentage               REAL,
    CorruptedPercentage        REAL,
    LostPercentage             REAL,
    OKCount                    INTEGER,
    CorruptedCount             INTEGER,
    LostCount                  INTEGER,
    AliveNodesCount            INTEGER,
    DeadNodesCount             INTEGER,
	TimerExpired			   BOOLEAN 
);`

	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
	flags.VPrintln("Table", tableName, "created successfully")
}

func writeToDB(db *sql.DB) error {

	query := `
	INSERT INTO ` + tableName + ` (
		ExperimentID, Algorithm, TimeDuration, Rounds, TotalMessages,
		MaxSentFromNode, MaxReceivedByNode,
		OKPercentage, CorruptedPercentage, LostPercentage,
		OKCount, CorruptedCount, LostCount,
		AliveNodesCount, DeadNodesCount, TimerExpired
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		Summary.ExperimentID,
		Summary.Algorithm,
		Summary.Time,
		Summary.Rounds,
		Summary.TotalMessages,
		Summary.MaxSentFromNode,
		Summary.MaxReceivedByNode,
		Summary.OKPercentage,
		Summary.CorruptedPercentage,
		Summary.LostPercentage,
		Summary.OKCount,
		Summary.CorruptedCount,
		Summary.LostCount,
		Summary.AliveNodesCount,
		Summary.DeadNodesCount,
		Summary.TimerExpired,
	)

	if err != nil {
		log.Printf("Failed to insert %s: %v", tableName, err)
	}
	return err
}

func getTimeDuration(db *sql.DB, experimentID int, algo string) (time.Duration, error) {
	var minTimeStr, maxTimeStr string

	query := `
		SELECT 
			MIN(Time) as min, 
			MAX(Time) as max
		FROM ` + algo + `
		WHERE ExperimentID = ?;
	`

	err := db.QueryRow(query, experimentID).Scan(&minTimeStr, &maxTimeStr)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}

	// Parse the times assuming format with nanoseconds
	const layout = "2006-01-02 15:04:05.999999999"
	minTime, err := time.Parse(layout, minTimeStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse min time: %w", err)
	}

	maxTime, err := time.Parse(layout, maxTimeStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse max time: %w", err)
	}

	return maxTime.Sub(minTime), nil
}

func getRoundsCount(algo string) int {

	switch algo {
	case "Broadcast":
		return 1
	case "Singlecast":
		return 1
	case "Multicast":
		return flags.Exper.MulticastDomains
	default:
		// For Gossip, information is send during simulation
		return Summary.Rounds
	}
}

func getTotalMessages(db *sql.DB, experimentID int, algo string) int {
	var totalMessages int
	var messageType string
	query := `
		SELECT MessageType, COUNT(*) AS count
		FROM ` + algo + `
		WHERE MessageType IS NOT NULL AND MessageType != '' AND ExperimentID = ?
		GROUP BY MessageType
		ORDER BY count DESC
		LIMIT 1;
	`

	err := db.QueryRow(query, experimentID).Scan(&messageType, &totalMessages)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf(ErrNoMessagesFmt, experimentID, algo)
			return 0 // No messages found
		}
	}
	fmt.Println("Total messages for experiment", experimentID, "in algorithm", algo, ":", totalMessages)
	return totalMessages
}

func getMaxSentFromNode(db *sql.DB, experimentID int, algo string) int {
	var MaxSentFromNode int
	var SenderID int
	query := `
		SELECT SenderID, COUNT(*) AS count
		FROM ` + algo + `
		WHERE MessageType = 'Send' and ExperimentID = ?
		GROUP BY SenderID
		ORDER BY count DESC
		LIMIT 1;
	`

	err := db.QueryRow(query, experimentID).Scan(&SenderID, &MaxSentFromNode)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf(ErrNoMessagesFmt, experimentID, algo)
			return 0 // No messages found
		}
	}
	return MaxSentFromNode
}

func getMaxReceivedByNode(db *sql.DB, experimentID int, algo string) int {
	var MaxReceivedByNode int
	var SenderID int
	query := `
		SELECT SenderID, COUNT(*) AS count
		FROM ` + algo + `
		WHERE MessageType = 'Receive' and ExperimentID = ?
		GROUP BY SenderID
		ORDER BY count DESC
		LIMIT 1;
	`

	err := db.QueryRow(query, experimentID).Scan(&SenderID, &MaxReceivedByNode)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf(ErrNoMessagesFmt, experimentID, algo)
			return 0 // No messages found
		}
	}
	return MaxReceivedByNode
}

func getDB(nodes []*node.Node) {
	Summary.OKCount = 0
	Summary.CorruptedCount = 0
	Summary.LostCount = 0
	for _, msg := range nodes {
		switch msg.DB.Data {
		case "OK":
			Summary.OKCount++
		case "corrupted":
			Summary.CorruptedCount++
		case "lost":
			Summary.LostCount++
		case "":
			Summary.LostCount++ // If no data, consider it lost
		}
	}
}

func getAliveAndDeadNodesCount(nodes []*node.Node) {
	Summary.AliveNodesCount = 0
	Summary.DeadNodesCount = 0
	for _, n := range nodes {
		if n.Alive {
			Summary.AliveNodesCount++
		} else {
			Summary.DeadNodesCount++
		}
	}
}

func getPercentages() {
	Summary.OKPercentage = float64(Summary.OKCount) / float64(flags.Exper.NodeCount) * 100
	Summary.CorruptedPercentage = float64(Summary.CorruptedCount) / float64(flags.Exper.NodeCount) * 100
	Summary.LostPercentage = float64(Summary.LostCount) / float64(flags.Exper.NodeCount) * 100
}
