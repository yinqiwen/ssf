package ssf

import "errors"

var ErrNoNode = errors.New("No cluster node to emit msg")
var ErrSSFDisconnect = errors.New("SSF disconnect")
var ErrNoClusterName = errors.New("No cluster name defined")
var ErrCommandTimeout = errors.New("Processor command timeout")
