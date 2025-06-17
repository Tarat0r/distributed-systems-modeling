package dissemination

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/analyze"
	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/network"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
)

type GossipMode string

const (
	GossipPush     GossipMode = "Push"
	GossipPull     GossipMode = "Pull"
	GossipPushPull GossipMode = "PushPull"
)

func Gossip(nodes []*node.Node, simulator *network.Simulator, fanout int, mode GossipMode, ready chan bool) {
	var wgGossip sync.WaitGroup
	rootNode := nodes[0]
	resetMsgID()

	respChans := make([]chan node.Message, len(nodes))
	for i := range respChans {
		respChans[i] = make(chan node.Message, len(nodes)/2)
	}

	msg := node.Message{
		SenderID:  rootNode.ID,
		Data:      "OK",
		MessageID: nextMsgID(),
	}
	rootNode.DB = msg

	// Открываем CSV файлы для каждого узла
	writers, csvFiles := openCSV(nodes)

	// START message
	metrics.AddExperimentStartTime()

	round := 0
	for {
		round++

		if gossipFinished(nodes, round) {
			break
		}

		for _, n := range nodes {
			for range fanout {
				wgGossip.Add(1)
				go gossipSend(n, simulator, mode, &wgGossip, respChans, writers)
			}
		}
		wgGossip.Wait() // ждем, пока все сообщения будут отправлены

		// ждем ответ от всех узлов
		for k, n := range nodes {
			select {
			case resp := <-respChans[k]:
				flags.VPrintln("Got response:", resp)

			case <-time.After(50 * time.Millisecond):
				flags.VPrintln("No response received from node", n.ID)
			}
		}
		// time.Sleep(10 * time.Millisecond)
		wgGossip.Wait() // ждем, пока все горутины завершатся
	}

	printResult(nodes)

	fmt.Println("All multicast messages sent, flushing CSV files...")
	for i := range writers {
		writers[i].Mutex.Lock()
		writers[i].Writer.Flush()
		writers[i].Mutex.Unlock()
		csvFiles[i].Close()
	}

	println("✅ All nodes received the message in round", round)
	analyze.Summary.Rounds = round

	ready <- true
}

func gossipSend(sender *node.Node, simulator *network.Simulator, mode GossipMode, wgGossip *sync.WaitGroup, respChans []chan node.Message, writers []*node.SafeWriter) {
	defer wgGossip.Done()

	receiver := getRandomPeer(sender)

	msgS := node.Message{
		SenderID:     sender.ID,
		Data:         sender.DB.Data,
		MessageID:    sender.DB.MessageID,
		ResponseChan: respChans[sender.ID],
	}

	msgR := node.Message{
		SenderID:     receiver.ID,
		Data:         receiver.DB.Data,
		MessageID:    receiver.DB.MessageID,
		ResponseChan: respChans[receiver.ID],
	}

	if receiver == nil {
		msgS.Data = "lost"
	}

	flags.VPrintln("Node", sender.ID, "is sending a message to", receiver.ID, "msg: ", msgS)
	switch mode {

	case GossipPush:

		if msgS.Data == "lost" || msgS.Data == "" {
			return
		}
		if err := node.WriteToCSV(writers[sender.ID], sender, &msgS, "Send"); err != nil {
			fmt.Println("Error writing to CSV:", err)
			return
		}
		wgGossip.Add(1)
		go simulator.Send(sender, receiver, msgS, wgGossip)

	case GossipPull:
		if msgR.Data == "OK" || msgR.Data == "corrupted" {
			return
		}
		if err := node.WriteToCSV(writers[sender.ID], receiver, &msgS, "Send"); err != nil {
			fmt.Println("Error writing to CSV PushPull:", err)
		}
		wgGossip.Add(1)
		go simulator.Send(sender, receiver, msgS, wgGossip)

	case GossipPushPull:
		time.Sleep(10 * time.Millisecond) // небольшая задержка для симуляции реального времени
		err := node.WriteToCSV(writers[sender.ID], sender, &msgS, "Send")
		err = node.WriteToCSV(writers[receiver.ID], receiver, &msgR, "Send")
		time.Sleep(10 * time.Millisecond) // небольшая задержка для симуляции реального времени
		if err != nil {
			fmt.Println("Error writing to CSV:", err)
		}
		wgGossip.Add(2)
		go simulator.Send(sender, receiver, msgS, wgGossip)
		go simulator.Send(receiver, sender, msgR, wgGossip)
	}

}

func getRandomPeer(n *node.Node) *node.Node {
	if len(n.Peers) == 0 {
		return nil
	}
	for {
		p := n.Peers[rand.Intn(len(n.Peers))]
		if p.ID != n.ID {
			return p
		}
	}
}

func printResult(nodes []*node.Node) {
	if !flags.Flags.Verbose {
		return
	}
	fmt.Println("========== Gossip Result ==========")
	for _, n := range nodes {
		if n.DB.Data != "" {
			fmt.Println("Node", n.ID, "received message:", n.DB.Data)
		} else {
			fmt.Println("Node", n.ID, "did not receive any message")
		}
	}
	fmt.Println("====================================")
}

func gossipFinished(nodes []*node.Node, round int) bool {
	// Count how many nodes received the message
	receivedCount := 0
	alive := 0
	for _, n := range nodes {
		flags.VPrintf("----- Node %d received message: %s\n", n.ID, n.DB.Data)
		if n.DB.Data == "OK" || n.DB.Data == "corrupted" {
			receivedCount++
		}
		if n.Alive {
			alive++
		}
	}

	fmt.Println("==========================Round:", round, "| Informed nodes:", receivedCount, "/", alive, "Alives")

	return receivedCount >= alive
}

func openCSV(nodes []*node.Node) ([]*node.SafeWriter, []*os.File) {
	var writers = make([]*node.SafeWriter, len(nodes))
	var csvFiles = make([]*os.File, len(nodes))
	for i, sender := range nodes {
		file, err := os.OpenFile("metrics/metrics_node_sender"+fmt.Sprintf("%d", sender.ID)+".csv", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening CSV file:", err)
			return nil, nil
		}
		// Add header if file is empty
		info, _ := file.Stat()
		if info.Size() == 0 {
			w := csv.NewWriter(file)
			w.Write([]string{"Experiment ID", "Time", "Node ID", "Alive Node", "Sender ID", "Receiver ID", "Messages Data", "Message ID", "Node DB", "Message Type"})
			w.Flush()
		}

		writers[i] = &node.SafeWriter{Writer: csv.NewWriter(file)}
		csvFiles[i] = file
	}

	return writers, csvFiles
}
