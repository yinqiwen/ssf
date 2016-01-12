package ssf

import "errors"

var ErrNoNode = errors.New("No cluster node to emit msg")
var ErrNoProcessor = errors.New("No processor to dispatch msg")
var ErrProcessorDisconnect = errors.New("Processor disconnect")
var ErrNoClusterName = errors.New("No cluster name defined")
var ErrRPCTimeout = errors.New("Processor RPC timeout")
var ErrEmptyProto = errors.New("Empty proto message")
