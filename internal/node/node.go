package node

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/fatih/color"
)

// Node represents a single node in a distributed system simulation.
type Node struct {
	ID int
	// Unique identifier for the node.

	Alive bool
	// Indicates whether the node is operational (true) or has failed (false).

	Peers []*Node
	// List of direct neighboring nodes (connections).
	// Each peer is a pointer to another Node, allowing direct communication
	// by sending messages to their Incoming channel.

	Incoming chan Message
	// Channel through which the node receives incoming messages asynchronously.
	// Other nodes send Message objects into this channel.

	DB Message
	// Database to store messages received by the node.
}

type Message struct {
	SenderID     int
	Data         string
	MessageID    int
	ResponseChan chan Message
}

type SafeWriter struct {
	Writer *csv.Writer
	Mutex  sync.Mutex
}

func (n *Node) Run() {

	csvFile, err := os.OpenFile("metrics/metrics_node_"+fmt.Sprintf("%d", n.ID)+".csv", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	writer := &SafeWriter{Writer: csv.NewWriter(csvFile)}

	defer csvFile.Close()

	for {
		select {
		case msg := <-n.Incoming:
			// обработка сообщения
			// Записываем в CSV
			err = WriteToCSV(writer, n, &msg, "Receive")
			if err != nil {
				fmt.Println("Error writing to CSV:", err)
				return
			}

			// Если узел не жив, игнорируем сообщение
			if n.Alive == false {
				color.Red("Node %d is NOT alive, ignoring message: %v\n", n.ID, msg)
				continue
			}

			if msg.Data == "lost" {
				color.Red("Node %d didn't received a message: %v\n", n.ID, msg)
				continue
			}

			if msg.Data == "corrupted" {
				color.Yellow("Node %d is alive, msg: %v\n", n.ID, msg)
			} else {
				color.Green("Node %d is alive, msg: %v\n", n.ID, msg)
			}

			if msg.MessageID > n.DB.MessageID {
				color.Green("Node %d received a new message with ID: (%d > %d), processing it\n", n.ID, msg.MessageID, n.DB.MessageID)
				n.DB = msg // сохраняем сообщение в базе данных узла
			} else {
				color.Yellow("Node %d received a message with an old ID: (%d < %d), ignoring it\n", n.ID, msg.MessageID, n.DB.MessageID)
			}

			// отправляем сообщение обратно в канал Incoming
			ResMsg := Message{
				SenderID:  n.ID,          // Устанавливаем ID отправителя
				Data:      msg.Data,      // Устанавливаем данные сообщения
				MessageID: msg.MessageID, // Сохраняем ID сообщения
			}

			msg.ResponseChan <- ResMsg // отправляем сообщение обратно в канал, если нужно
		}
	}
}

func NewCluster(size int) []*Node {
	nodes := make([]*Node, size)
	for i := range nodes {
		nodes[i] = &Node{
			ID:       i,
			Alive:    true,
			Incoming: make(chan Message, flags.Exper.NodeCount),
		}
	}
	// Связываем узлы в Peers
	for _, n := range nodes {
		n.Peers = nodes // в простом случае все знают всех
	}
	return nodes
}

// CopyAlive copies the Alive state from old nodes to new nodes by matching IDs.
func CopyAlive(nodes []*Node, aliveMask []*bool) error {
	if len(nodes) != len(aliveMask) {
		return fmt.Errorf("length of nodes and aliveMask must be the same")
	}

	for i, n := range nodes {
		n.Alive = *aliveMask[i]
	}
	return nil
}

func InitCSVFiles(N int) error {
	// Создаём CSV файлы для каждого узла
	for i := 0; i < N; i++ {
		file, err := os.Create("metrics/metrics_node_" + fmt.Sprintf("%d", i) + ".csv")
		if err != nil {
			fmt.Println("Error creating CSV file:", err)
			return err
		}
		defer file.Close()

		// Write headers to CSV
		writer := csv.NewWriter(file) // Create a CSV writer
		err = writer.Write([]string{"Experiment ID", "Time", "Node ID", "Alive Node", "Sender ID", "Receiver ID", "Messages Data", "Message ID", "Node DB", "Message Type"})
		if err != nil {
			fmt.Println("Error writing CSV header:", err)
			return err
		}

		writer.Flush() // Ensure the data is written to the file
		if err := writer.Error(); err != nil {
			fmt.Println("Error flushing CSV writer:", err)
			return err
		}
	}
	return nil
}

func WriteToCSV(writer *SafeWriter, n *Node, msg *Message, msgType string) error {

	writer.Mutex.Lock()
	defer writer.Mutex.Unlock()

	err := writer.Writer.Write([]string{
		fmt.Sprintf("%d", flags.Exper.ID),                  // Experiment ID
		time.Now().Format("2006-01-02 15:04:05.000000000"), // Time
		fmt.Sprintf("%v", n.ID),                            // Node ID
		fmt.Sprintf("%v", n.Alive),                         // Number of alive nodes
		fmt.Sprintf("%d", msg.SenderID),                    // Sender ID
		fmt.Sprintf("%d", n.ID),                            // Receiver ID
		fmt.Sprintf("%s", msg.Data),                        // Message data
		fmt.Sprintf("%d", msg.MessageID),                   // Message data
		fmt.Sprintf("%s", n.DB.Data),                       // Node DB
		fmt.Sprintf("%s", msgType),                         // Type of message (Send, Receive, etc.)
	})
	if err != nil {
		fmt.Println("Error writing to CSV:", err)
		return err
	}
	writer.Writer.Flush() // Ensure the data is written to the file
	if err := writer.Writer.Error(); err != nil {
		fmt.Println("Error flushing CSV writer:", err)
		return err
	}
	return nil
}
