package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var workerList []*rpc.Client

func subscribe(factoryAddress string) (err error) {
	fmt.Println("Subscription request")
	client, err := rpc.Dial("tcp", factoryAddress)
	if err == nil {
		workerList = append(workerList, client)
	} else {
		fmt.Println("Error subscribing ", factoryAddress)
		fmt.Println(err)
		return err
	}
	return
}

func singleCall(client *rpc.Client, req stubs.DistributorRequest) [][]byte {
	fmt.Println("Called:")
	response := new(stubs.WorkerResponse)
	err := client.Call(stubs.WorkerTurns, req, response)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return response.World
}

func pluralCall(req stubs.DistributorRequest) [][]byte {
	fmt.Println("Called:")
	//todo copy code in doublecall
	return req.WorldEven
}

type Broker struct{}
type AllTurns struct{}

func (b *Broker) Subscribe(req stubs.Subscription, res *stubs.StatusReport) (err error) {
	err = subscribe(req.FactoryAddress)
	if err != nil {
		res.Message = "Error during subscription"
	}
	return err
}

func (t *AllTurns) AllTurns(req stubs.DistributorRequest, res *stubs.DistributorResponse) (err error) {
	fmt.Println(req.Turns)
	if len(workerList) <= 0 {
		return errors.New("No Workers")
	}
	if len(workerList) == 1 {
		singleCall(workerList[0], req)
	} else {
		pluralCall(req)
	}
	return
}
func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	err := rpc.Register(&Broker{})
	if err != nil {
		fmt.Println("err")
		return
	}
	err = rpc.Register(&AllTurns{})
	if err != nil {
		fmt.Println("err")
		return
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
