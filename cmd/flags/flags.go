package flags

import (
	"flag"
	"fmt"
)

type Config struct {
	Verbose  bool
	RemoveDB bool
}

var Flags = &Config{}

type Experiment struct {
	ID                    int
	Timer                 int
	NodeCount             int
	MulticastDomains      int
	GossipFanOut          int
	DelayMean             int
	AliveProbability      float64
	LossProbability       float64
	CorruptionProbability float64
}

var Exper Experiment // Экспериментальные параметры

// RegisterFlags binds command-line flags to the config fields
func RegisterFlags() {

	flag.BoolVar(&Flags.Verbose, "v", false, "enable verbose output")
	flag.BoolVar(&Flags.Verbose, "verbose", false, "same as -v")

	flag.BoolVar(&Flags.RemoveDB, "r", false, "remove database before running the simulation")
	flag.BoolVar(&Flags.RemoveDB, "remove-db", false, "same as -r")

	flag.IntVar(&Exper.ID, "id", 0, "experiment ID")
	flag.IntVar(&Exper.Timer, "timer", 0, "Wait time in seconds before starting the simulation (0 for no wait)")
	flag.IntVar(&Exper.NodeCount, "nodes", 10, "number of nodes")
	flag.IntVar(&Exper.MulticastDomains, "domains", 3, "number of domains for multicast simulation")
	flag.IntVar(&Exper.GossipFanOut, "fanout", 1, "Gossip fan-out factor (number of nodes to which each node sends messages)")
	flag.IntVar(&Exper.DelayMean, "delay", 20, "network delay")
	flag.Float64Var(&Exper.AliveProbability, "alive", 1.0, "probability node is alive")
	flag.Float64Var(&Exper.LossProbability, "loss", 0.03, "message loss probability")
	flag.Float64Var(&Exper.CorruptionProbability, "corrupt", 0.05, "message corruption probability")

}

func VPrintln(args ...any) {
	if Flags.Verbose {
		fmt.Println(args...)
	}
}

func VPrintf(format string, args ...any) {
	if Flags.Verbose {
		fmt.Printf(format, args...)
	}
}
