step1: replace the node url in `eth_test.go`

step2: run `TestClient_NewTxTraceHandler()`

result: for BSC both `pending` && `latest` option works, for `ETH` only `latest` works.