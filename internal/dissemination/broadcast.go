package dissemination

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/network"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
	"github.com/fatih/color"
)

var globalMsgID int
var msgIDMu sync.Mutex

func Broadcast(nodes []*node.Node, simulator *network.Simulator, ready chan bool) {
	sender := nodes[0] // стартовый узел
	sender.DB = node.Message{Data: "OK"}

	csvFile, err := os.OpenFile("metrics/metrics_node_"+fmt.Sprintf("%d", sender.ID)+".csv", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	writer := &node.SafeWriter{}
	writer.Writer = csv.NewWriter(csvFile)
	defer csvFile.Close()

	var wg sync.WaitGroup
	resetMsgID()

	metrics.AddExperimentStartTime()

	wg.Add(1)                                                             // добавляем в WaitGroup, чтобы дождаться завершения отправки сообщений
	go BroadcastFromNode(sender, simulator, writer, &wg, nextMsgID, "OK") // отправляем сообщения от стартового узла
	wg.Wait()                                                             // ждем завершения отправки сообщений
	ready <- true                                                         // сигнализируем, что сообщение отправлено
}

func BroadcastFromNode(
	sender *node.Node,
	simulator *network.Simulator,
	writer *node.SafeWriter,
	wg *sync.WaitGroup,
	nextMsgID func() int,
	msgData string,
) []string {
	var responses []string
	color.HiMagenta("Broadcasting message from node " + fmt.Sprintf("%d", sender.ID))

	defer wg.Done()
	respChans := make([]chan node.Message, len(sender.Peers))
	for i := range respChans {
		respChans[i] = make(chan node.Message, len(sender.Peers)/2)
	}

	for k, reciver := range sender.Peers {
		if reciver.ID == sender.ID {
			continue
		}

		msg := node.Message{
			SenderID:     sender.ID,
			Data:         msgData,
			MessageID:    nextMsgID(),
			ResponseChan: respChans[k],
		}

		flags.VPrintln(sender.ID, "->", reciver.ID, "msg:", msg)

		wg.Add(1)
		go simulator.Send(sender, reciver, msg, wg)

		if err := node.WriteToCSV(writer, sender, &msg, "Send"); err != nil {
			fmt.Println("Error writing to CSV:", err)
			return nil
		}
	}

	// ждем ответ от всех узлов
	for k, reciver := range sender.Peers {
		if reciver.ID == sender.ID {
			continue
		}

		select {
		case resp := <-respChans[k]:
			flags.VPrintln("Got response:", resp)
			responses = append(responses, resp.Data)

		case <-time.After(50 * time.Millisecond):
			responses = append(responses, "lost")
			flags.VPrintln("No response received from node", reciver.ID)
		}
	}

	flags.VPrintln("++++++++++++ Broadcast completed Loop ++++++++++++")

	// wg.Wait()
	return responses
}

func nextMsgID() int {
	msgIDMu.Lock()
	defer msgIDMu.Unlock()
	globalMsgID++
	return globalMsgID
}

func resetMsgID() {
	msgIDMu.Lock()
	defer msgIDMu.Unlock()
	globalMsgID = 0
}
