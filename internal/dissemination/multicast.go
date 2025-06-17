package dissemination

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/network"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
)

func Multicast(nodes []*node.Node, simulator *network.Simulator, multicastDomains int, ready chan bool) {
	var senders = make([]*node.Node, multicastDomains)

	if multicastDomains < 2 {
		fmt.Println("Error: need at least 2 multicast domains")
		return
	}

	var i int
	for i = range multicastDomains {
		senders[i] = nodes[i]
		senders[i].Peers = make([]*node.Node, 0, len(nodes)/multicastDomains) // инициализируем слайс для пиров
	}

	senders[0].Peers = append(senders[0].Peers, senders[1:]...) // первый узел получает всех остальных в пирах
	senders[0].DB = node.Message{Data: "OK"}                    // инициализируем базу данных первого узла
	var j int = 0
	for i = i + 1; i < len(nodes); i++ {
		senders[j].Peers = append(senders[j].Peers, nodes[i])
		j = (j + 1) % multicastDomains
	}

	flags.VPrintln(PrintSenderPeers(senders))

	// Открываем CSV файлы для каждого узла
	var writers = make([]*node.SafeWriter, len(senders))
	var csvFiles = make([]*os.File, len(senders))
	var err error
	for i, sender := range senders {
		csvFiles[i], err = os.OpenFile("metrics/metrics_node_"+fmt.Sprintf("%d", sender.ID)+".csv", os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening CSV file:", err)
			return
		}
		writers[i] = &node.SafeWriter{Writer: csv.NewWriter(csvFiles[i])}
	}

	var wgMultiCast sync.WaitGroup
	resetMsgID()
	msgData := "OK"

	metrics.AddExperimentStartTime()

	// Отправляем сообщения от стартового узла
	wgMultiCast.Add(1)
	responses := make([]string, len(senders)-1) // инициализируем слайс для ответов
	responses = BroadcastFromNode(senders[0], simulator, writers[0], &wgMultiCast, nextMsgID, msgData)
	responses = append([]string{"Root_sender"}, responses...) // добавляем ответ от первого узла
	wgMultiCast.Wait()

	for k, sender := range senders[1:] { // начинаем с 1, т.к. 0 уже отправил сообщения
		wgMultiCast.Add(1)
		fmt.Println(responses)
		fmt.Println("k:", k+1, "Sender ID:", sender.ID)
		BroadcastFromNode(sender, simulator, writers[k+1], &wgMultiCast, nextMsgID, responses[k+1])
	}

	fmt.Println("Waiting for multicast messages to be sent...")
	wgMultiCast.Wait() // ждём, пока все сообщения будут отправлены
	fmt.Println("All multicast messages sent, flushing CSV files...")
	for i := range writers {
		writers[i].Mutex.Lock()
		writers[i].Writer.Flush()
		writers[i].Mutex.Unlock()
		csvFiles[i].Close()
	}

	ready <- true // сигнализируем, что сообщение отправлено
}

func PrintSenderPeers(senders []*node.Node) string {
	var output []string
	for _, n := range senders {
		output = append(output, fmt.Sprintf("Sender[%d] -> Peers: ", n.ID))

		for _, p := range n.Peers {
			output = append(output, fmt.Sprintf("%d,", p.ID))
		}
		output = append(output, "\n")
	}
	return strings.Join(output, "")
}
