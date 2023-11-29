package stubs

import "net/rpc"

var AllTurns = "Broker.AllTurns"
var WorkerTurns = "WorkerTurns.WorkerTurnsSingle"
var WorkersTurns = "WorkerTurns.WorkerTurnsPlural"

//Broker and Distributor stubs
type DistributorResponse struct {
	World [][]byte
}
type DistributorRequest struct {
	WorldEven   [][]byte
	ImageWidth  int
	ImageHeight int
	Turns       int
}

//Broker and Worker stubs
type WorkerResponse struct {
	World [][]byte
}
type WorkerRequest struct {
	client      *rpc.Client
	WorldEven   [][]byte
	ImageWidth  int
	ImageHeight int
	Turns       int
}

//Worker and worker stubs
type RowSwap struct {
	Row []byte
}

//worker and broker registration stubs
type Subscription struct {
	FactoryAddress string
}

type StatusReport struct {
	Message string
}
