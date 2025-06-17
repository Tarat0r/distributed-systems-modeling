package dissemination

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/network"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
)

func Singlecast(nodes []*node.Node, simulator *network.Simulator, ready chan bool) {
	var wg sync.WaitGroup
	start := nodes[0] // стартовый узел

	start.DB = node.Message{Data: "OK"}

	csvFile, err := os.OpenFile("metrics/metrics_node_"+fmt.Sprintf("%d", start.ID)+".csv", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening CSV file:", err)
		return
	}
	writer := &node.SafeWriter{}
	writer.Writer = csv.NewWriter(csvFile)
	defer csvFile.Close()

	metrics.AddExperimentStartTime()

	msgID := 1
	msgData := "OK"
outer:
	for j, sender := range nodes {
		if sender.ID == len(nodes)-1 {
			continue // пропускаем последний узел, чтобы не отправлять ему сообщение
		}
		respChan := make(chan node.Message)
		msg := node.Message{
			SenderID:     sender.ID,
			Data:         msgData,
			MessageID:    msgID,
			ResponseChan: respChan,
		}
		msgID++
		reciver := nodes[j+1]

		wg.Add(1)
		go simulator.Send(sender, reciver, msg, &wg)

		err = node.WriteToCSV(writer, reciver, &msg, "Send") // Записываем в CSV
		if err != nil {
			fmt.Println("Error writing to CSV:", err)
			return
		}

		select {
		case resp := <-respChan:
			fmt.Println("Got response:", resp)
			if resp.Data == "corrupted" {
				msgData = "corrupted" // если сообщение повреждено, меняем данные сообщения
			}

		case <-time.After(50 * time.Millisecond): // ждем ответа 50 мс
			fmt.Println("Timeout waiting for node", reciver.ID)
			break outer
		}
	}
	wg.Wait()     // ждем завершения всех горутин
	ready <- true // сигнализируем, что сообщение отправлено
}
