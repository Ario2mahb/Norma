package nodemon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Fantom-foundation/Norma/driver"
	mon "github.com/Fantom-foundation/Norma/driver/monitoring"
	opera "github.com/Fantom-foundation/Norma/driver/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// NodeBlockHeight collects a per-node time series of its current block height.
var NodeBlockHeight = mon.Metric[mon.Node, mon.TimeSeries[int]]{
	Name:        "NodeBlockHeight",
	Description: "The block height of nodes at various times.",
}

func init() {
	if err := mon.RegisterSource(NodeBlockHeight, NewNodeBlockHeightSource); err != nil {
		panic(fmt.Sprintf("failed to register metric source: %v", err))
	}
}

// NewNumNodesSource creates a new data source periodically collecting data on
// the block height at various nodes over time.
func NewNodeBlockHeightSource(network driver.Network) mon.Source[mon.Node, mon.TimeSeries[int]] {
	return newNodeBlockHeightSource(network, time.Second)
}

func newNodeBlockHeightSource(network driver.Network, period time.Duration) mon.Source[mon.Node, mon.TimeSeries[int]] {
	return newPeriodicNodeDataSource[int](NodeBlockHeight, network, period, &blockProgressSensorFactory{})
}

type blockProgressSensorFactory struct{}

func (f *blockProgressSensorFactory) CreateSensor(node driver.Node) (Sensor[int], error) {
	url := node.GetHttpServiceUrl(&opera.OperaRpcService)
	if url == nil {
		return nil, fmt.Errorf("node does not export an RPC server")
	}
	rpcClient, err := rpc.DialContext(context.Background(), string(*url))
	if err != nil {
		return nil, err
	}
	return &blockProgressSensor{rpcClient}, nil
}

type blockProgressSensor struct {
	rpcClient *rpc.Client
}

func (s *blockProgressSensor) ReadValue() (int, error) {
	var blockNumber string
	err := s.rpcClient.Call(&blockNumber, "eth_blockNumber")
	if err != nil {
		return 0, err
	}
	blockNumber = strings.TrimPrefix(blockNumber, "0x")
	value, err := strconv.ParseInt(blockNumber, 16, 32)
	if err != nil {
		return 0, err
	}
	return int(value), nil
}