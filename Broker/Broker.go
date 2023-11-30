package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var (
	workerList   []*rpc.Client
	workerListmx sync.RWMutex
)

func active() {
	i := 0
	for {
		i++
		time.Sleep(10 * time.Second)
		fmt.Println(i, "workers:", len(workerList))
	}
}
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

func pluralCalls(req stubs.DistributorRequest) [][]byte {
	fmt.Println("Called:")
	workers := len(workerList)
	heightsplit := req.ImageHeight / workers
	if req.ImageHeight%workers != 0 {
		heightsplit += 1
	}
	out := make([]chan [][]uint8, workers)
	for i := range out {
		out[i] = make(chan [][]uint8)
	}
	newRequest := stubs.WorkerRequest{}
	newRequest.ImageWidth = req.ImageWidth
	for i, v := range workerList {
		startY := i * heightsplit
		endY := i + 1*heightsplit
		if endY > req.ImageHeight {
			endY = req.ImageHeight
		}

		newRequest.ImageHeight = endY - startY
		newRequest.WorldEven = req.WorldEven[startY:endY]

		if i < workers {
			newRequest.Client = workerList[i+1]
		} else {
			newRequest.Client = workerList[i+1]
		}
		go pluralCall(newRequest, out[i], v)
	}
	var finishedWorld [][]byte
	for i := 0; i < workers; i++ {
		part := <-out[i]
		finishedWorld = append(finishedWorld, part...)
	}
	return finishedWorld
}

func pluralCall(req stubs.WorkerRequest, rtrn chan [][]uint8, client *rpc.Client) {
	response := new(stubs.WorkerResponse)
	err := client.Call(stubs.WorkerTurns, req, response)
	if err != nil {
		fmt.Println(err)
		return
	}
	rtrn <- response.World
}

type Broker struct{}

func (b *Broker) Subscribe(req stubs.Subscription, res *stubs.StatusReport) (err error) {
	workerListmx.Lock()
	err = subscribe(req.FactoryAddress)
	if err != nil {
		res.Message = "Error during subscription"
	}
	workerListmx.Unlock()
	return err
}

type AllTurns struct{}

func (t *AllTurns) AllTurns(req stubs.DistributorRequest, res *stubs.DistributorResponse) (err error) {
	fmt.Println(req.Turns)
	workerListmx.Lock()

	if len(workerList) <= 0 {
		return errors.New("No Workers")
	}
	if len(workerList) == 1 {
		res.World = singleCall(workerList[0], req)
	}
	if len(workerList) > 1 {
		res.World = pluralCalls(req)
	}
	workerListmx.Unlock()
	return
}
func main() {
	pAddr := flag.String("port", "8031", "Port to listen on")
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
	go active()
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(listener)
	rpc.Accept(listener)
}
