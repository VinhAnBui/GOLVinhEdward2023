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

//receives an array of bytes from ioInput
func receiveWorld(ioInput <-chan uint8, p Params, events chan<- Event) [][]byte {
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
				events <- CellFlipped{0, util.Cell{X: x, Y: y}}
			}
			world[y][x] = val
		}
	}
	return world
}

//sends world to ioOutput
func sendWorld(ioOutput chan<- uint8, world [][]byte) {
	for _, row := range world {
		for _, v := range row {
			ioOutput <- v
		}
	}
}

//GOL logic

func numAdjacentLiving(world [][]byte, x int, y int, imageWidth int, imageHeight int) int {
	var numAdjacentLiving = 0
	if (x > imageWidth) || (y > imageHeight) || (0 > x) || (0 > y) {
		panic("Co-ordinates out of range!")
	}
	var cellToLeftX, cellToRightX, cellAboveY, cellBelowY int

	//get cells next to target cell, wrapping around if necessary
	if x == 0 {
		cellToLeftX = imageWidth - 1
	} else {
		cellToLeftX = x - 1
	}
	if x == imageWidth-1 {
		cellToRightX = 0
	} else {
		cellToRightX = x + 1
	}
	if y == 0 {
		cellBelowY = imageHeight - 1
	} else {
		cellBelowY = y - 1
	}
	if y == imageHeight-1 {
		cellAboveY = 0
	} else {
		cellAboveY = y + 1
	}

	//increment numAdjacentLiving for each adjacent living cell
	if world[cellToLeftX][y] == 255 {
		numAdjacentLiving++
	}
	if world[cellToLeftX][cellAboveY] == 255 {
		numAdjacentLiving++
	}
	if world[x][cellAboveY] == 255 {
		numAdjacentLiving++
	}
	if world[cellToRightX][cellAboveY] == 255 {
		numAdjacentLiving++
	}
	if world[cellToRightX][y] == 255 {
		numAdjacentLiving++
	}
	if world[cellToRightX][cellBelowY] == 255 {
		numAdjacentLiving++
	}
	if world[x][cellBelowY] == 255 {
		numAdjacentLiving++
	}
	if world[cellToLeftX][cellBelowY] == 255 {
		numAdjacentLiving++
	}

	return numAdjacentLiving
}

func partWorld(turn, startY, endY, startX, endX int, world, newWorld [][]byte, params Params, group *sync.WaitGroup, events chan<- Event) {
	stageConverter(turn, startY, endY, startX, endX, world, newWorld, params, events)
	group.Done()
}
func stageConverter(turn, startY, endY, startX, endX int, world, newWorld [][]byte, params Params, events chan<- Event) {

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			var numAdjacentLiving = numAdjacentLiving(world, x, y, params.ImageWidth, params.ImageHeight)
			switch {
			case ((numAdjacentLiving < 2) || (numAdjacentLiving > 3)) && (world[x][y] == 255):

				newWorld[x][y] = 0
			case (numAdjacentLiving == 3) && (world[x][y] == 0):
				newWorld[x][y] = 255
			default:
				newWorld[x][y] = world[x][y]
			}
			if newWorld[y][x] != world[y][x] {
				events <- CellFlipped{turn, util.Cell{X: x, Y: y}}
			}

		}
	}
}
func newWorld(turn int, world, newWorld [][]byte, p Params, mutex *sync.Mutex, events chan<- Event) {
	mutex.Lock()
	if p.Threads == 1 {
		stageConverter(turn, 0, p.ImageHeight, 0, p.ImageWidth, world, newWorld, p, events)
	} else {
		var wg sync.WaitGroup
		heightSplit := p.ImageHeight / p.Threads
		if p.ImageHeight%p.Threads != 0 {
			heightSplit += 1
		}
		for i := 0; i < p.Threads; i++ {
			wg.Add(1)
			startY := heightSplit * i
			endY := heightSplit * (i + 1)
			if endY > p.ImageHeight {
				endY = p.ImageHeight
			}
			go partWorld(turn, startY, endY, 0, p.ImageWidth, world, newWorld, p, &wg, events)
		}
		wg.Wait()
	}
	mutex.Unlock()
}

/*
//useful for debugging reasons
func printWorld(world [][]byte) {
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
*/
func aliveCells(world [][]byte) []util.Cell {

	var aliveCells []util.Cell
	for yi, row := range world {
		for xi, value := range row {
			if value == 255 {
				aliveCells = append(aliveCells, util.Cell{X: xi, Y: yi})
			}
		}
	}
	return aliveCells
}
func every2Seconds(oddWorld, evenWorld [][]byte, turn *int, oddMutex, evenMutex, turnLock *sync.Mutex, events chan<- Event) {
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
					sendWorld(c.ioOutput, worldEven)
				} else {
					sendWorld(c.ioOutput, worldOdd)
				}
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
			case 'q':
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(*turn)
				if *turn%2 == 0 {
					sendWorld(c.ioOutput, worldEven)
				} else {
					sendWorld(c.ioOutput, worldOdd)
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	c.ioCommand <- ioInput
	c.ioFilename <- filenameInput(p)
	worldEven := receiveWorld(c.ioInput, p, c.events)

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
	go every2Seconds(worldOdd, worldEven, &turn, oddMutex, EvenMutex, turnLock, c.events)
	go waitKeypress(&turn, worldOdd, worldEven, turnLock, c, p)
	for turn < p.Turns {
		turnLock.Lock()
		if turn%2 == 0 {
			newWorld(turn, worldEven, worldOdd, p, EvenMutex, c.events)
		} else {
			newWorld(turn, worldOdd, worldEven, p, oddMutex, c.events)
		}
		c.events <- TurnComplete{turn}
		turn++
		turnLock.Unlock()
	}

	c.ioCommand <- ioOutput
	c.ioFilename <- filenameOutput(p)
	if turn%2 != 0 {
		sendWorld(c.ioOutput, worldOdd)
		c.events <- FinalTurnComplete{turn, aliveCells(worldOdd)}
	} else {
		sendWorld(c.ioOutput, worldEven)
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
