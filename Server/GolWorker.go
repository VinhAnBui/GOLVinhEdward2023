package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var (
	topRowIn   [][]byte
	topRowInmx sync.Mutex

	topRowOut   [][]byte
	topRowOutmx sync.Mutex
)

//GOL logic
//count3x3 counts how many live cells there are in a 3 x 3 area centered around some x y
func count3x3(grid [][]byte, x, y, height, width int) int {
	//fmt.Println(x, y)
	count := 0
	for xi := -1; xi < 2; xi++ {
		xi2 := x + xi
		//fmt.Println("xi:", xi)
		xi2 = edgeReset(xi2, width)
		//fmt.Println("xi2:", xi2)
		for yi := -1; yi < 2; yi++ {
			//fmt.Println("yi:", yi)
			yi2 := yi + y
			yi2 = edgeReset(yi2, height)
			//fmt.Println("yi:", yi2)
			if grid[yi2][xi2] == 255 {
				count += 1
			}
		}
	}
	return count
}

//if out of array loops the value back around again
func edgeReset(i int, max int) int {
	if i < 0 {
		return max - 1
	}
	if i >= max {
		return 0
	}
	return i
}

//cell value should return the value of a cell given its count
func cellValue(count int, cellValue byte) byte {
	switch count {
	case 3:
		return 255
	case 4:
		if cellValue != 0 {
			return 255
		}
	}
	return 0
}
func stageConverter(startY, endY, startX, endX, height, width int, world, newWorld [][]byte) {
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, height, width), world[y][x])
		}
	}
}

func callRowExchange(row []byte, client *rpc.Client) []byte {
	req := stubs.RowSwap{}
	req.Row = row
	res := stubs.RowSwap{}
	err := client.Call(stubs.RowExchange, req, res)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return res.Row
}

type WorkerTurns struct{}

func (t *WorkerTurns) WorkerTurnsSingle(req stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
	fmt.Println("aaaaaaaaaaaaaa")
	fmt.Println(req.Turns)
	worldEven := req.WorldEven
	worldOdd := make([][]byte, req.ImageHeight)
	for i := range worldOdd {
		worldOdd[i] = make([]byte, req.ImageWidth)
	}
	//var turnLock = &sync.Mutex{}
	turn := 0
	for turn < req.Turns {
		//turnLock.Lock()
		if turn%2 == 0 {
			stageConverter(0, req.ImageHeight, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldEven, worldOdd)
		} else {
			stageConverter(0, req.ImageHeight, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldOdd, worldEven)
		}
		turn++
		//turnLock.Unlock()
	}
	//deadlock occurs without this line
	//turnLock.Lock()
	if turn%2 == 0 {
		res.World = worldEven
	} else {
		res.World = worldOdd
	}
	return
}
func (t *WorkerTurns) WorkerTurnsPlural(req stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
	topRowOutmx.Lock()
	fmt.Println(req.Turns)
	worldEven := req.WorldEven
	worldOdd := make([][]byte, req.ImageHeight)
	for i := range worldOdd {
		worldOdd[i] = make([]byte, req.ImageWidth)
	}
	//var turnLock = &sync.Mutex{}
	turn := 0
	for turn < req.Turns {
		//turnLock.Lock()
		if turn%2 == 0 {
			stageConverter(1, req.ImageHeight-1, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldEven, worldOdd)
			topRowOutmx.Unlock()
			worldOdd[req.ImageHeight-1] = callRowExchange(worldOdd[1], req.Client)
			topRowInmx.Lock()
			worldOdd[0] = topRowIn[0]
			topRowIn = topRowIn[1:]
			topRowInmx.Unlock()
		} else {
			stageConverter(1, req.ImageHeight-1, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldOdd, worldEven)
			topRowOutmx.Unlock()
			worldOdd[req.ImageHeight-1] = callRowExchange(worldEven[1], req.Client) //exchanges bottom row
			topRowInmx.Lock()
			worldOdd[0] = topRowIn[0] //exchanges top row
			topRowIn = topRowIn[1:]
			topRowInmx.Unlock()
		}

		turn++
		topRowOutmx.Lock()
	}
	//deadlock occurs without this line
	//turnLock.Lock()
	if turn%2 == 0 {
		res.World = worldEven
	} else {
		res.World = worldOdd
	}

	return
}

type RowExchange struct{}

func (t *RowExchange) rowExchange(req stubs.RowSwap, res *stubs.RowSwap) (err error) {
	topRowInmx.Lock()
	topRowIn = append(topRowIn, req.Row)
	topRowInmx.Unlock()
	topRowOutmx.Lock()
	res.Row = topRowOut[0]
	topRowOut = topRowOut[1:]
	topRowOutmx.Unlock()
	return
}
func active() {
	i := 0
	for {
		i++
		time.Sleep(10 * time.Second)
		fmt.Println("alive", i)
	}
}
func main() {
	// Parse command-line arguments to get the port
	pAddr := flag.String("port", "8031", "port to listen on")
	flag.Parse()
	fmt.Println(pAddr)
	// Register the RPC service
	rpc.Register(&WorkerTurns{})
	rpc.Register(&RowExchange{})
	fmt.Println(pAddr, 2)
	// Listen for incoming connections on the specified port
	listener, err := net.Listen("tcp", ":"+*pAddr)
	fmt.Println(pAddr, 3)
	if err != nil {
		// Handle the error and exit or log it
		fmt.Println("Error listening:", err)
		return
	}
	fmt.Println(pAddr, 4)
	defer listener.Close()
	go active()
	rpc.Accept(listener)
	fmt.Println(pAddr, 5)
}
