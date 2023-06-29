package monitoring

import (
	"github.com/Fantom-foundation/Norma/driver"
	"github.com/golang/mock/gomock"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestLogsParsersImplements(t *testing.T) {
	var inst NodeLogDispatcher
	var _ NodeLogProvider = &inst
}

func TestRegisterLogParser(t *testing.T) {
	ctrl := gomock.NewController(t)
	net := driver.NewMockNetwork(ctrl)

	node1 := driver.NewMockNode(ctrl)
	node2 := driver.NewMockNode(ctrl)
	node3 := driver.NewMockNode(ctrl)

	node1.EXPECT().GetLabel().AnyTimes().Return(string(Node1TestId))
	node2.EXPECT().GetLabel().AnyTimes().Return(string(Node2TestId))
	node3.EXPECT().GetLabel().AnyTimes().Return(string(Node3TestId))

	node1.EXPECT().StreamLog().AnyTimes().Return(io.NopCloser(strings.NewReader(Node1TestLog)), nil)
	node2.EXPECT().StreamLog().AnyTimes().Return(io.NopCloser(strings.NewReader(Node2TestLog)), nil)
	node3.EXPECT().StreamLog().AnyTimes().Return(io.NopCloser(strings.NewReader(Node3TestLog)), nil)

	// simulate existing nodes
	net.EXPECT().RegisterListener(gomock.Any())
	net.EXPECT().GetActiveNodes().AnyTimes().Return([]driver.Node{})

	reg := NewNodeLogDispatcher(net)
	ch := make(chan Node, 10)
	listener := &testBlockNodeListener{data: map[Node][]Block{}, ch: ch}
	reg.RegisterLogListener(listener)

	// simulate added node
	reg.AfterNodeCreation(node1)
	reg.AfterNodeCreation(node2)
	reg.AfterNodeCreation(node3)

	// drain 3 nodes from the channel
	for _, node := range []Node{<-ch, <-ch, <-ch} {
		got := listener.getBlocks(node)
		want := NodeBlockTestData[node]
		blockEqual(t, node, got, want)
	}

	if reg.getNumNodes() != 3 {
		t.Errorf("wrong size")
	}

	if len(reg.getNodes()) != 3 {
		t.Errorf("wrong number of iterations")
	}
}

type testBlockNodeListener struct {
	data     map[Node][]Block
	dataLock sync.Mutex
	ch       chan Node
}

func blockEqual(t *testing.T, node Node, got, want []Block) {
	if len(got) != len(want) {
		t.Errorf("wrong blocks collected for Node %v: %v != %v", node, got, want)
	}

	for i, b := range got {
		if want[i].Height != b.Height || want[i].Txs != b.Txs || want[i].GasUsed != b.GasUsed {
			t.Errorf("wrong blocks collected for Node %v: %v != %v", node, want[i], b)
		}
	}
}

func (l *testBlockNodeListener) OnBlock(node Node, b Block) {
	l.dataLock.Lock()
	defer l.dataLock.Unlock()

	// send uniq nodes
	if _, exists := l.data[node]; !exists {
		l.ch <- node
	}

	// count in only non-empty blocks
	if b.Height > 0 {
		l.data[node] = append(l.data[node], b)
	}
}

func (l *testBlockNodeListener) getBlocks(node Node) []Block {
	l.dataLock.Lock()
	defer l.dataLock.Unlock()

	return l.data[node]
}
