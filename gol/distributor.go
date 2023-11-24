package gol

import (
	"fmt"
	"strconv"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
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

//GOL logic
//count3x3 counts how many live cells there are in a 3 x 3 area centered around some x y
func count3x3(grid [][]byte, x, y int, params Params) int {
	//fmt.Println(x, y)
	count := 0
	for xi := -1; xi < 2; xi++ {
		xi2 := x + xi
		//fmt.Println("xi:", xi)
		xi2 = edgereset(xi2, params.ImageWidth)
		//fmt.Println("xi2:", xi2)
		for yi := -1; yi < 2; yi++ {
			//fmt.Println("yi:", yi)
			yi2 := yi + y
			yi2 = edgereset(yi2, params.ImageHeight)
			//fmt.Println("yi:", yi2)
			if grid[yi2][xi2] == 255 {
				count += 1
			}
		}
	}
	return count
}

//if out of array loops the value back around again
func edgereset(i int, max int) int {
	if i < 0 {
		return (max - 1)
	}
	if i >= max {
		return 0
	}
	return i
}

//cell value should return the value of a cell given its count
func cellValue(count int, cellvalue byte) byte {
	switch count {
	case 3:
		return 255
	case 4:
		if cellvalue != 0 {
			return 255
		}
	}
	return 0
}
func partWorld(turn, startY, endY, startX, endX int, world, newWorld [][]byte, params Params, group *sync.WaitGroup, events chan<- Event) {
	stageConverter(turn, startY, endY, startX, endX, world, newWorld, params, events)
	group.Done()
}
func stageConverter(turn, startY, endY, startX, endX int, world, newWorld [][]byte, params Params, events chan<- Event) {

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, params), world[y][x])
			if newWorld[y][x] != world[y][x] {
				events <- CellFlipped{turn, util.Cell{x, y}}
			}

		}
	}
}
func newworld(turn int, world, newWorld [][]byte, p Params, mutex *sync.Mutex, events chan<- Event) {
	mutex.Lock()
	if p.Threads == 1 {
		stageConverter(turn, 0, p.ImageHeight, 0, p.ImageWidth, world, newWorld, p, events)
	} else {
		var wg sync.WaitGroup
		heightsplit := p.ImageHeight / p.Threads
		if p.ImageHeight%p.Threads != 0 {
			heightsplit += 1
		}
		for i := 0; i < p.Threads; i++ {
			wg.Add(1)
			startY := heightsplit * i
			endY := heightsplit * (i + 1)
			if endY > p.ImageHeight {
				endY = p.ImageHeight
			}
			go partWorld(turn, startY, endY, 0, p.ImageWidth, world, newWorld, p, &wg, events)
		}
		wg.Wait()
	}
	mutex.Unlock()
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	c.ioCommand <- ioInput
	c.ioFilename <- filenameInput(p)
	worldEven := recieveworld(c.ioInput, p, c.events)

	worldOdd := make([][]byte, p.ImageHeight)
	for i := range worldOdd {
		worldOdd[i] = make([]byte, p.ImageWidth)
	}

	//controls access to the odd or even matrix
	var oddMutex = &sync.Mutex{}
	var EvenMutex = &sync.Mutex{}
	//controls access to turn
	var turnLock = &sync.Mutex{}

	turn := 0
	go every2seconds(worldOdd, worldEven, &turn, oddMutex, EvenMutex, turnLock, c.events)
	for turn < p.Turns {
		turnLock.Lock()
		if turn%2 == 0 {
			newworld(turn, worldEven, worldOdd, p, EvenMutex, c.events)
		} else {
			newworld(turn, worldOdd, worldEven, p, oddMutex, c.events)
		}
		c.events <- TurnComplete{turn}
		turn++
		turnLock.Unlock()
	}
	c.ioCommand <- ioOutput
	c.ioFilename <- filenameOutput(p)
	if turn%2 != 0 {
		sendsworld(c.ioOutput, worldOdd)
		c.events <- FinalTurnComplete{turn, aliveCells(worldOdd)}
	} else {
		sendsworld(c.ioOutput, worldEven)
		c.events <- FinalTurnComplete{turn, aliveCells(worldEven)}
	}

	//deadlock occurs without this line
	turnLock.Lock()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.

	close(c.events)
}
