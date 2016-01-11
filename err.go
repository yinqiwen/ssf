package ssf

import "errors"

var ErrNoNode = errors.New("No cluster node to emit msg")
var ErrNoProcessor = errors.New("No processor to dispatch msg")
var ErrSSFDisconnect = errors.New("SSF disconnect")
var ErrNoClusterName = errors.New("No cluster name defined")
var ErrRPCTimeout = errors.New("Processor RPC timeout")
