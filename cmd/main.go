package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/analyze"
	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/dissemination"
	"github.com/Tarat0r/distributed-systems-modeling/internal/metrics"
	"github.com/Tarat0r/distributed-systems-modeling/internal/network"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
	"github.com/fatih/color"
)

var bold = color.New(color.Bold, color.BgCyan)

const errSimulationPreparation = "Error during simulation preparation"

func init() {
	flags.RegisterFlags()
	flag.Parse()
}

func main() {

	// Remove old DB if remove-db flag is set
	if flags.Flags.RemoveDB {
		if err := metrics.RemoveDB(); err != nil {
			fmt.Println("Error removing database records:", err)
			return
		}
		fmt.Println("Old database records removed successfully")
	}

	bold.Println("\n==== Preparing Simulation ====")

	// flags.Exper = flags.Experiment{
	// 	ID:                    0, // ID эксперимента, можно использовать для сохранения результатов в БД
	// 	Timer:                 0,
	// 	NodeCount:             500,
	// 	MulticastDomains:      4,
	// 	GossipFanOut:          2,    // количество узлов, на которые каждый узел отправляет сообщения в Gossip
	// 	DelayMean:             20,   // среднее время задержки в миллисекундах
	// 	AliveProbability:      0.8,  // вероятность того, что узел Alive
	// 	LossProbability:       0.21, // вероятность потери сообщения
	// 	CorruptionProbability: 0.22, // вероятность потери или порчи сообщения
	// }

	networkSimulator := network.NewSimulator(flags.Exper)
	ready := make(chan bool)

	aliveMask := network.SetAlives(flags.Exper) // устанавливаем Alive матрицу для узлов с вероятностью 0.8

	broadcastSimulation(flags.Exper, aliveMask, networkSimulator, ready)                            // запускаем симуляцию рассылки
	singlecastSimulation(flags.Exper, aliveMask, networkSimulator, ready)                           // запускаем симуляцию однокастовой рассылки
	multicastSimulation(flags.Exper, aliveMask, networkSimulator, ready)                            // запускаем симуляцию однокастовой рассылки
	gossipSimulation(flags.Exper, aliveMask, networkSimulator, ready, dissemination.GossipPull)     // запускаем симуляцию Push Gossip{
	gossipSimulation(flags.Exper, aliveMask, networkSimulator, ready, dissemination.GossipPush)     // запускаем симуляцию Push Gossip{
	gossipSimulation(flags.Exper, aliveMask, networkSimulator, ready, dissemination.GossipPushPull) // запускаем симуляцию Push Gossip{

	bold.Println("\n==== Simulation Finished ====")

	metrics.WriteExperimentToDB()

	printMemStats()

}

func printMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func simulationPreparation(N int, aliveMaskPtrs []*bool) ([]*node.Node, error) {
	nodes := node.NewCluster(N) // создаём узлов

	node.CopyAlive(nodes, aliveMaskPtrs)

	err := node.InitCSVFiles(N)
	if err != nil {
		fmt.Println("Error initializing CSV files:", err)
		return nil, err
	}

	for _, n := range nodes {
		go n.Run()
		flags.VPrintln("Node", n.ID, "started")

	}

	return nodes, nil
}

func broadcastSimulation(exper flags.Experiment, aliveMask []*bool, networkSimulator *network.Simulator, ready chan bool) {

	nodes, err := simulationPreparation(exper.NodeCount, aliveMask) // создаём узлы и запускаем их
	if err != nil {
		fmt.Println(errSimulationPreparation)
		return
	}
	bold.Println("\n==== Starting Broadcast Simulation ====")

	go dissemination.Broadcast(nodes, networkSimulator, ready)
	// metrics.StartMonitoring(nodes)
	waitWithTimer(ready)
	if flags.Flags.Verbose {
		color.HiMagenta("\n==== Aggregating Broadcast Metrics ====")
	}
	metrics.AgregateToDB("Broadcast")
	analyze.Analyze(nodes, "Broadcast")

	color.HiMagenta("Broadcast Simulation completed")
}

func singlecastSimulation(exper flags.Experiment, aliveMask []*bool, networkSimulator *network.Simulator, ready chan bool) {
	bold.Println("\n==== Starting Singlecast Simulation ====")

	nodes, err := simulationPreparation(exper.NodeCount, aliveMask) // создаём узлы и запускаем их
	if err != nil {
		fmt.Println(errSimulationPreparation)
		return
	}

	go dissemination.Singlecast(nodes, networkSimulator, ready)
	waitWithTimer(ready)
	if flags.Flags.Verbose {
		color.HiMagenta("\n==== Aggregating Singlecast Metrics ====")
	}
	metrics.AgregateToDB("Singlecast")
	analyze.Analyze(nodes, "Singlecast")
	color.HiMagenta("Singlecast Simulation completed")
}

func multicastSimulation(exper flags.Experiment, aliveMask []*bool, networkSimulator *network.Simulator, ready chan bool) {
	bold.Println("\n==== Starting Multicast Simulation ====")

	nodes, err := simulationPreparation(exper.NodeCount, aliveMask) // создаём узлы и запускаем их
	if err != nil {
		fmt.Println(errSimulationPreparation)
		return
	}

	go dissemination.Multicast(nodes, networkSimulator, exper.MulticastDomains, ready)
	waitWithTimer(ready)
	if flags.Flags.Verbose {
		color.HiMagenta("\n==== Aggregating Multicast Metrics ====")
	}
	metrics.AgregateToDB("Multicast")
	analyze.Analyze(nodes, "Multicast")
	color.HiMagenta("Multicast Simulation completed")
}

func gossipSimulation(exper flags.Experiment, aliveMask []*bool, networkSimulator *network.Simulator, ready chan bool, mode dissemination.GossipMode) {
	bold.Printf("\n==== Starting Gossip %v Simulation ====\n", mode)

	nodes, err := simulationPreparation(exper.NodeCount, aliveMask)
	if err != nil {
		fmt.Println(errSimulationPreparation)
		return
	}

	go dissemination.Gossip(nodes, networkSimulator, exper.GossipFanOut, mode, ready)
	waitWithTimer(ready)
	if flags.Flags.Verbose {
		color.HiMagenta("\n==== Aggregating Gossip Metrics ====")
	}

	metrics.AgregateToDB(fmt.Sprintf("Gossip%v", mode))
	analyze.Analyze(nodes, fmt.Sprintf("Gossip%v", mode))
	color.HiMagenta("Gossip Simulation completed")
}

func waitWithTimer(ready chan bool) {
	timer := flags.Exper.Timer // получаем значение таймера из флагов
	if timer <= 0 {
		//wait infinitely if timer is 0 or negative
		<-ready
		analyze.Summary.TimerExpired = false
		return
	}

	select {
	case <-ready:
		analyze.Summary.TimerExpired = false

	case <-time.After(time.Duration(timer) * time.Second):
		analyze.Summary.TimerExpired = true
		color.HiRed("Timer expired!")
	}
}
