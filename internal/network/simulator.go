package network

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Tarat0r/distributed-systems-modeling/cmd/flags"
	"github.com/Tarat0r/distributed-systems-modeling/internal/node"
)

type Simulator struct {
	LossProbability       float64
	DelayMean             int
	CorruptionProbability float64
}

func NewSimulator(exper flags.Experiment) *Simulator {
	return &Simulator{exper.LossProbability, exper.DelayMean, exper.CorruptionProbability}
}

func (s *Simulator) Send(sender *node.Node, receiver *node.Node, msg node.Message, wg *sync.WaitGroup) {
	defer wg.Done()

	delay := time.Duration(rand.Intn(2*s.DelayMean)) * time.Millisecond
	time.Sleep(delay)

	if rand.Float64() < s.CorruptionProbability {
		msg.Data = "corrupted" // corrupted message
	}

	if rand.Float64() < s.LossProbability {
		msg.Data = "lost" // lost message
	}

	receiver.Incoming <- msg
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Sending message from node", sender.ID, "to node", receiver.ID)
}

func SetAlives(exper flags.Experiment) []*bool {
	aliveMask := make([]bool, exper.NodeCount, exper.NodeCount)
	for i := range exper.NodeCount {
		if rand.Float64() < exper.AliveProbability {
			aliveMask[i] = true
		} else {
			aliveMask[i] = false
		}
	}
	aliveMaskPtrs := make([]*bool, len(aliveMask))
	for i := range aliveMask {
		aliveMaskPtrs[i] = &aliveMask[i]
	}
	return aliveMaskPtrs
}
