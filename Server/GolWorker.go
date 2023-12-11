package main

import (
	"flag"
	"fmt"
	"github.com/ChrisGora/semaphore"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var (
	topRowIn         = newBuffer(3)
	topRowInmx       sync.Mutex
	inWorkAvailable  = semaphore.Init(3, 0)
	topRowOut        = newBuffer(3)
	outWorkAvailable = semaphore.Init(3, 0)
	topRowOutmx      sync.Mutex

	turnLock sync.Mutex
	command  chan int
	world    chan [][]byte
	turn     chan int
)

type buffer struct {
	b                 [][]byte
	size, read, write int
}

func newBuffer(size int) buffer {
	return buffer{
		b:     make([][]byte, size),
		size:  size,
		read:  0,
		write: 0,
	}
}

func (buffer *buffer) get() []byte {
	x := buffer.b[buffer.read]
	fmt.Println("Get\t\t", x, "\t", buffer)
	buffer.read = (buffer.read + 1) % len(buffer.b)
	return x
}

func (buffer *buffer) put(x []byte) {
	buffer.b[buffer.write] = x
	fmt.Println("Put\t\t", x, "\t", buffer)
	buffer.write = (buffer.write + 1) % len(buffer.b)
}

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

//rpc stuff
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
func getOutboundIP() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Println(err)

		}
	}(conn)
	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	fmt.Println(localAddr)
	return localAddr
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
		turnLock.Lock()
		if turn%2 == 0 {
			stageConverter(0, req.ImageHeight, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldEven, worldOdd)
		} else {
			stageConverter(0, req.ImageHeight, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldOdd, worldEven)
		}
		turn++
		turnLock.Unlock()
	}
	//deadlock occurs without this line
	turnLock.Lock()
	if turn%2 == 0 {
		res.World = worldEven
	} else {
		res.World = worldOdd
	}
	return
}
func (t *WorkerTurns) WorkerTurnsPlural(req stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
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
			topRowOutmx.Lock()
			topRowOut.put(worldOdd[1])
			outWorkAvailable.Post()
			topRowOutmx.Unlock()
			worldOdd[req.ImageHeight-1] = callRowExchange(worldOdd[1], req.Client)
			inWorkAvailable.Wait()
			topRowInmx.Lock()
			worldOdd[0] = topRowIn.get()
		} else {
			stageConverter(1, req.ImageHeight-1, 0, req.ImageWidth, req.ImageHeight, req.ImageWidth, worldOdd, worldEven)
			topRowOutmx.Lock()
			topRowOut.put(worldEven[1])
			outWorkAvailable.Post()
			topRowOutmx.Unlock()
			worldEven[req.ImageHeight-1] = callRowExchange(worldEven[1], req.Client)
			inWorkAvailable.Wait()
			topRowInmx.Lock()
			worldEven[0] = topRowIn.get() //exchanges top row
		}
		topRowInmx.Unlock()
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

type RowExchange struct{}

func (t *RowExchange) RowExchange(req stubs.RowSwap, res *stubs.RowSwap) (err error) {
	topRowInmx.Lock()
	topRowIn.put(req.Row)
	inWorkAvailable.Post()
	topRowInmx.Unlock()
	outWorkAvailable.Wait()
	topRowOutmx.Lock()
	res.Row = topRowOut.get()
	topRowOutmx.Unlock()
	return
}
func active() {
	i := 0
	for {
		i++
		time.Sleep(10 * time.Second)
		fmt.Println(i, "workers:")
	}
}
func main() {
	// Parse command-line arguments to get the port
	brokerAddr := flag.String("broker", "3.94.78.254:8030", "Address of broker instance")
	pAddr := flag.String("port", "8050", "Port to listen on")
	flag.Parse()
	fmt.Println(brokerAddr)
	client, _ := rpc.Dial("tcp", *brokerAddr)
	fmt.Println("1")
	status := new(stubs.StatusReport)
	// Register the RPC service
	err := rpc.Register(&WorkerTurns{})
	if err != nil {
		fmt.Println(err)
	}
	err = rpc.Register(&RowExchange{})
	if err != nil {
		fmt.Println(err)
	}
	//registers itself as a worker to broker
	err = client.Call(stubs.Subscribe, stubs.Subscription{FactoryAddress: getOutboundIP() + ":" + *pAddr}, status)
	if err != nil {
		fmt.Println(err)
	}
	// Listen for incoming connections on the specified port
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		// Handle the error and exit or log it
		fmt.Println("Error listening:", err)
		return
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(listener)
	go active()
	rpc.Accept(listener)
}
