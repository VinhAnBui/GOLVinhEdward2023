package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	keyPresses <-chan rune
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

//generates the filename for inputting
func filenameInput(p Params) string {
	s := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth)
	fmt.Println(s)
	return s
}

//generates the filename for outputting
func filenameOutput(p Params) string {
	s := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	fmt.Println(s)
	return s
}

//recieves an array of bytes from ioInput
func recieveworld(ioInput <-chan uint8, p Params, events chan<- Event) [][]byte {
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-ioInput
			//if val != 0 {
			//	fmt.Println(x, y)
			//}
			if val != 0 {
				events <- CellFlipped{0, util.Cell{x, y}}
			}
			world[y][x] = val
		}
	}
	return world
}

//sends world to ioOutput
func sendsworld(ioOutput chan<- uint8, world [][]byte) {
	for _, row := range world {
		for _, v := range row {
			ioOutput <- v
			if v != 0 {
				//fmt.Println("send coords:", strconv.Itoa(x), strconv.Itoa(y))
			}
		}
	}
}

//useful for debugging reasons
func printworld(world [][]byte) {
	for _, v := range world {
		var b []byte
		for _, v2 := range v {
			if v2 == 255 {
				b = append(b, 1)
			} else {
				b = append(b, v2)
			}
		}
		fmt.Println(b)
	}
	println("")
}
func aliveCells(world [][]byte) []util.Cell {

	var aliveCells []util.Cell
	for yi, row := range world {
		for xi, value := range row {
			if value == 255 {
				aliveCells = append(aliveCells, util.Cell{xi, yi})
			}
		}
	}
	return aliveCells
}
func every2seconds(oddWorld, evenWorld [][]byte, turn *int, oddMutex, evenMutex, turnLock *sync.Mutex, events chan<- Event) {
	for {
		time.Sleep(2 * time.Second)
		turnLock.Lock()
		if *turn%2 == 0 {
			evenMutex.Lock()
			events <- AliveCellsCount{*turn, aliveCount(evenWorld)}
			evenMutex.Unlock()
		} else {
			oddMutex.Lock()
			events <- AliveCellsCount{*turn, aliveCount(oddWorld)}
			oddMutex.Unlock()
		}
		turnLock.Unlock()
	}
}
func aliveCount(world [][]byte) int {
	count := 0
	for _, row := range world {
		for _, value := range row {
			if value != 0 {
				count += 1
			}
		}
	}
	return count
}
func waitKeypress(turn *int, worldOdd, worldEven [][]byte, turnLock *sync.Mutex, c distributorChannels, p Params) {
	for {
		select {
		// Block and wait for requests from the distributor
		case command := <-c.keyPresses:
			turnLock.Lock()
			switch command {
			case 's':
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(*turn)
				if *turn%2 == 0 {
					sendsworld(c.ioOutput, worldEven)
				} else {
					sendsworld(c.ioOutput, worldOdd)
				}
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
			case 'q':
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(*turn)
				if *turn%2 == 0 {
					sendsworld(c.ioOutput, worldEven)
				} else {
					sendsworld(c.ioOutput, worldOdd)
				}
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
				c.events <- StateChange{*turn, Quitting}
				close(c.events)
			case 'p':
				fmt.Println("Turn:", *turn)
				<-c.keyPresses
				fmt.Println("Continuing")
			}
			turnLock.Unlock()
		}
	}
}
func makeCall(client *rpc.Client, worldEven [][]byte, p Params) [][]byte {
	fmt.Println("Called:")
	request := stubs.DistributorRequest{WorldEven: worldEven, ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns}
	response := new(stubs.DistributorResponse)
	err := client.Call(stubs.AllTurns, request, response)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	printworld(response.World)
	fmt.Println("All turns complete")
	return response.World
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	c.ioCommand <- ioInput
	c.ioFilename <- filenameInput(p)
	worldEven := recieveworld(c.ioInput, p, c.events)

	//controls access to the odd or even matrix
	//var oddMutex = &sync.Mutex{}
	//var EvenMutex = &sync.Mutex{}
	//controls access to turn
	//var turnLock = &sync.Mutex{}
	//go every2seconds(worldOdd, worldEven, &turn, oddMutex, EvenMutex, turnLock, c.events)
	//go waitKeypress(&turn, worldOdd, worldEven, turnLock, c, p)

	server := flag.String("server", "172.31.88.20:8040", "IP:port string to connect to as server")
	flag.Parse()
	client, err := rpc.Dial("tcp", *server)
	fmt.Println("ERROR IS: ", err)
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(client)
	world := makeCall(client, worldEven, p)

	//turnLock.Lock()
	c.ioCommand <- ioOutput
	c.ioFilename <- filenameOutput(p)
	sendsworld(c.ioOutput, world)
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- FinalTurnComplete{p.Turns, aliveCells(world)}
	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
