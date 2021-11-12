package eth

import "testing"

var (
	ethNode = "ws://"
	bscNode = "ws://"
)

func TestClient_NewTxTraceHandler(t *testing.T) {
	client, err := NewClient(bscNode)
	if err != nil {
		return
	}
	//client.NewTxTraceHandler("pending")
	client.NewTxTraceHandler("latest")
}
