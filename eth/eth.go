package eth

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"math/big"
	"time"
)

type (
	Client struct {
		EthClient *ethclient.Client
		RpcClient *rpc.Client
		NetworkId *big.Int
	}

	TraceResult struct {
		Calls   []TraceResult `json:"calls"`
		From    string        `json:"from"`
		Gas     string        `json:"gas"`
		GasUsed string        `json:"gasUsed"`
		Input   string        `json:"input"`
		Output  string        `json:"output"`
		Time    string        `json:"time"`
		To      string        `json:"to"`
		Type    string        `json:"type"`
		Value   string        `json:"value"`
	}
)

func NewClient(url string) (*Client, error) {
	client := new(Client)
	if ethClient, err := ethclient.Dial(url); err != nil {
		return nil, err
	} else {
		client.EthClient = ethClient
	}
	if rpcClient, err := rpc.Dial(url); err != nil {
		return nil, err
	} else {
		client.RpcClient = rpcClient
	}
	if networkId, err := client.EthClient.NetworkID(context.Background()); err != nil {
		return nil, err
	} else {
		client.NetworkId = networkId
	}
	return client, nil
}

func (client *Client) NewTxTraceHandler(opt string) error {
	ch := make(chan common.Hash, 2000)
	subscribe, err := client.RpcClient.EthSubscribe(context.Background(), ch, "newPendingTransactions")
	if err != nil {
		return err
	}
	for {
		select {
		case txHash := <-ch:
			go func() {
				tx, traceResult, err := client.TraceCall(txHash, opt)
				if err != nil {
					logrus.Error(txHash, err)
				}
				if tx != nil && traceResult != nil {
					logrus.Println(txHash, traceResult.From, traceResult.To)
				}
			}()
		case <-time.After(30 * time.Second):
			logrus.Error("timeout")
		case err := <-subscribe.Err():
			logrus.Error(err)
		}
	}
}

func (client *Client) TraceCall(txHash common.Hash, opt string) (tx *types.Transaction, traceResult *TraceResult, err error) {
	tx, _, err = client.EthClient.TransactionByHash(context.Background(), txHash)
	if err != nil && err != ethereum.NotFound {
		logrus.Error("TransactionByHash: ", err)
		return
	}
	if tx == nil {
		return
	}
	var result interface{}
	tracerOpt := "callTracer"
	arg, err := toCallArg(tx, client.NetworkId)
	if err != nil {
		return
	}
	if arg == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logConfig := vm.LogConfig{DisableStack: true, DisableStorage: true, Debug: true}
	err = client.RpcClient.CallContext(ctx, &result, "debug_traceCall", arg, opt, &tracers.TraceConfig{LogConfig: &logConfig, Tracer: &tracerOpt})
	if err != nil {
		return
	}
	traceResult = new(TraceResult)
	if err = mapstructure.Decode(result, traceResult); err != nil {
		return
	}
	return
}

func toCallArg(tx *types.Transaction, chainID *big.Int) (interface{}, error) {
	from, err := types.Sender(types.LatestSignerForChainID(chainID), tx)
	if err != nil {
		return nil, err
	}
	arg := map[string]interface{}{
		"from": from,
		"to":   tx.To(),
	}
	if len(tx.Data()) > 0 {
		arg["data"] = hexutil.Bytes(tx.Data())
	}
	if tx.Value() != nil {
		arg["value"] = (*hexutil.Big)(tx.Value())
	}
	if tx.Gas() != 0 {
		arg["gas"] = hexutil.Uint64(tx.Gas())
	}
	if tx.GasPrice() != nil {
		arg["gasPrice"] = (*hexutil.Big)(tx.GasPrice())
	}
	return arg, nil
}
