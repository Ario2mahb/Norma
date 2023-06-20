package nodemon

import (
	"fmt"
	"github.com/Fantom-foundation/Norma/driver"
	"github.com/Fantom-foundation/Norma/driver/monitoring"
	"github.com/golang/mock/gomock"
	"os"
	"testing"
	"time"
)

func TestSMASource(t *testing.T) {
	ctrl := gomock.NewController(t)

	net := driver.NewMockNetwork(ctrl)
	net.EXPECT().RegisterListener(gomock.Any()).AnyTimes()
	net.EXPECT().GetActiveNodes().Return([]driver.Node{}).AnyTimes()

	writer := monitoring.NewMockWriterChain(ctrl)
	writer.EXPECT().Add(gomock.Any()).AnyTimes()

	monitor := monitoring.NewMonitor(net, monitoring.MonitorConfig{}, writer)

	source := NewTransactionsThroughputSource(monitoring.NewMonitor(net, monitoring.MonitorConfig{}, writer))
	sf := sourceFactory[monitoring.Node, monitoring.BlockSeries[float32]]{TransactionsThroughput, source}
	if err := monitoring.InstallSource[monitoring.Node, monitoring.BlockSeries[float32]](monitor, &sf); err != nil {
		t.Fatalf("failed to install source: %v", err)
	}

	period := 2
	smaFactory := func(input monitoring.BlockSeries[float32]) monitoring.BlockSeries[float32] {
		return monitoring.NewSMASeries[monitoring.BlockNumber, float32](input, period)
	}
	TransactionThroughputSMA := monitoring.Metric[monitoring.Node, monitoring.BlockSeries[float32]]{
		Name:        fmt.Sprintf("TransactionThroughputSMA_%d", period),
		Description: "Transaction throughput standard moving average",
	}

	smaSource := newNodeBlockSeriesTransformation(monitor, TransactionThroughputSMA, TransactionsThroughput, smaFactory)

	// fill in original source with data
	for node, blocks := range monitoring.NodeBlockTestData {
		for _, block := range blocks {
			source.OnBlock(node, block)
		}
	}

	expected := map[monitoring.Node][]float32{
		monitoring.Node1TestId: {43.763676, 52.494083},
		monitoring.Node2TestId: {40.733196},
		monitoring.Node3TestId: {},
	}

	// test SMA computed for each node
	for node := range monitoring.NodeBlockTestData {
		series, exists := smaSource.GetData(node)
		if !exists {
			t.Errorf("series does not exist for subject: %v", node)
		}

		points := series.GetRange(monitoring.BlockNumber(0), monitoring.BlockNumber(1000))
		for i, block := range points {
			if got, want := block.Value, expected[node][i]; got != want {
				t.Errorf("data series contain unexpected value for: %v: %v != %v", node, got, want)
			}
		}

		if got, want := len(points), len(expected[node]); got != want {
			t.Errorf("number of points does not mathc: %d != %d", got, want)
		}
	}

	// test subjects present
	for _, node := range smaSource.GetSubjects() {
		if _, exists := monitoring.NodeBlockTestData[node]; !exists {
			t.Errorf("subject %v is not present", node)
		}
	}

	if got, want := len(smaSource.GetSubjects()), len(monitoring.NodeBlockTestData); got != want {
		t.Errorf("number of subjects does not mathc: %d != %d", got, want)
	}
}

func TestSMACsvExport(t *testing.T) {
	ctrl := gomock.NewController(t)

	net := driver.NewMockNetwork(ctrl)
	net.EXPECT().RegisterListener(gomock.Any()).AnyTimes()
	net.EXPECT().GetActiveNodes().AnyTimes().Return([]driver.Node{})

	csvFile, _ := os.CreateTemp(t.TempDir(), "file.csv")
	writer := monitoring.NewWriterChain(csvFile)

	monitor := monitoring.NewMonitor(net, monitoring.MonitorConfig{}, writer)
	source := NewTransactionsThroughputSource(monitoring.NewMonitor(net, monitoring.MonitorConfig{}, writer))
	sf := sourceFactory[monitoring.Node, monitoring.BlockSeries[float32]]{TransactionsThroughput, source}
	if err := monitoring.InstallSource[monitoring.Node, monitoring.BlockSeries[float32]](monitor, &sf); err != nil {
		t.Fatalf("failed to install source: %v", err)
	}

	period := 2
	smaFactory := func(input monitoring.BlockSeries[float32]) monitoring.BlockSeries[float32] {
		return monitoring.NewSMASeries[monitoring.BlockNumber, float32](input, period)
	}
	TransactionThroughputSMA := monitoring.Metric[monitoring.Node, monitoring.BlockSeries[float32]]{
		Name:        fmt.Sprintf("TransactionThroughputSMA_%d", period),
		Description: "Transaction throughput standard moving average",
	}

	newNodeBlockSeriesTransformation(monitor, TransactionThroughputSMA, TransactionsThroughput, smaFactory)

	seconds := time.Now().Unix()
	// time diff only 50ns
	source.OnBlock("A", monitoring.Block{Height: 10, Time: time.Unix(seconds, 0), Txs: 10})
	source.OnBlock("A", monitoring.Block{Height: 11, Time: time.Unix(seconds+1, 0), Txs: 10})
	_ = writer.Close()

	content, _ := os.ReadFile(csvFile.Name())
	if got, want := string(content),
		"TransactionsThroughput, network, A, , , 11, , 10\n"+
			"TransactionThroughputSMA_2, network, A, , , 11, , 10\n"; got != want {

		t.Errorf("unexpected export: %v != %v", got, want)
	}
}

type sourceFactory[S any, T any] struct {
	metric monitoring.Metric[S, T]
	source monitoring.Source[S, T]
}

func (f *sourceFactory[S, T]) GetMetric() monitoring.Metric[S, T] {
	return f.metric
}

func (f *sourceFactory[S, T]) CreateSource(*monitoring.Monitor) monitoring.Source[S, T] {
	return f.source
}