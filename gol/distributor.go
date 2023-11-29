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

type cell struct {
	x int
	y int
}
type cellResult struct {
	cell    cell
	isAlive bool
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

func cellResultWorker(world [][]byte, cells chan cell, imageWidth int, imageHeight int,
	cellResults chan cellResult, events chan<- Event, turn int, done chan bool, wg *sync.WaitGroup, newWorldStartTime time.Time) {

	var cellsProcessedTracker = 0 //remove after debugging

	for {
		select {
		case currentCell := <-cells:

			var cellToLeftX, cellToRightX, cellAboveY, cellBelowY int
			numAdjacentLiving := 0
			if currentCell.x == 0 {
				cellToLeftX = imageWidth - 1
			} else {
				cellToLeftX = currentCell.x - 1
			}

			if currentCell.x == imageWidth-1 {
				cellToRightX = 0
			} else {
				cellToRightX = currentCell.x + 1
			}

			if currentCell.y == 0 {
				cellBelowY = imageHeight - 1
			} else {
				cellBelowY = currentCell.y - 1
			}

			if currentCell.y == imageHeight-1 {
				cellAboveY = 0
			} else {
				cellAboveY = currentCell.y + 1
			}

			if world[cellToLeftX][currentCell.y] == 255 {
				numAdjacentLiving++
			}
			if world[cellToLeftX][cellAboveY] == 255 {
				numAdjacentLiving++
			}
			if world[currentCell.x][cellAboveY] == 255 {
				numAdjacentLiving++
			}
			if world[cellToRightX][cellAboveY] == 255 {
				numAdjacentLiving++
			}
			if world[cellToRightX][currentCell.y] == 255 {
				numAdjacentLiving++
			}
			if world[cellToRightX][cellBelowY] == 255 {
				numAdjacentLiving++
			}
			if world[currentCell.x][cellBelowY] == 255 {
				numAdjacentLiving++
			}
			if world[cellToLeftX][cellBelowY] == 255 {
				numAdjacentLiving++
			}

			switch {
			case ((numAdjacentLiving < 2) || (numAdjacentLiving > 3)) && world[currentCell.x][currentCell.y] == 255:

				cellResults <- cellResult{cell: currentCell, isAlive: false}
				events <- CellFlipped{turn, util.Cell{X: currentCell.x, Y: currentCell.y}}

			case (numAdjacentLiving == 3) && !(world[currentCell.x][currentCell.y] == 255):
				cellResults <- cellResult{cell: currentCell, isAlive: true}
				events <- CellFlipped{turn, util.Cell{X: currentCell.x, Y: currentCell.y}}
			default:
				cellResults <- cellResult{cell: currentCell, isAlive: world[currentCell.x][currentCell.y] == 255}
			}
			cellsProcessedTracker++
		case <-done:
			wg.Done()
			return
		}
	}

}

func newWorld(turn int, world [][]byte, p Params, events chan<- Event) [][]byte {

	start := time.Now()
	var wg sync.WaitGroup

	//queue for cellResults as output by workers
	cellResults := make(chan cellResult)

	//queue for cells that need to be solved
	cellQueue := make(chan cell)

	//contents of the channel doesn't matter - sending a value to it stops worker routines
	killSignal := make(chan bool)

	nextWorld := make([][]byte, p.ImageHeight)
	for i := range nextWorld {
		nextWorld[i] = make([]byte, p.ImageWidth)
	}

	go func() {
		var taskCount = 0
		for n := 0; n < len(world)*len(world[0]); n++ {
			i := <-cellResults

			if i.isAlive {
				nextWorld[i.cell.x][i.cell.y] = 255
			} else {
				nextWorld[i.cell.x][i.cell.y] = 0
			}
			taskCount++

		}
		close(killSignal)

	}()
	//create workers equal to the number of threads
	for i := 0; i < p.Threads; i++ {
		wg.Add(1)
		go cellResultWorker(world, cellQueue, p.ImageWidth, p.ImageHeight, cellResults, events, turn, killSignal, &wg, start)

	}

	for x := 0; x < p.ImageWidth; x++ {
		for y := 0; y < p.ImageWidth; y++ {
			go func(x, y int) {
				cellQueue <- cell{x: x, y: y}
			}(x, y)

		}
	}

	wg.Wait()
	return nextWorld

	//For each available thread, start a goroutine to first calculate the orthogonal adjacent cells
	//to a certain co-ordinate, then get the number of adjacent living ones, then return a slice like
	// [x, y, alive/dead]
	//read from that channel and set cells in newWorld according to results from it.
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
func updateAliveCellsCount(world [][]byte, turn *int, turnLock *sync.Mutex, events chan<- Event) {
	for {
		time.Sleep(2 * time.Second)
		turnLock.Lock()
		events <- AliveCellsCount{*turn, aliveCount(world)}
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
func waitKeypress(turn *int, world [][]byte, turnLock *sync.Mutex, c distributorChannels, p Params) {
	for {
		select {
		// Block and wait for requests from the distributor
		case command := <-c.keyPresses:
			turnLock.Lock()
			switch command {
			case 's':
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(*turn)
				sendWorld(c.ioOutput, world)
				c.ioCommand <- ioCheckIdle
				<-c.ioIdle
			case 'q':
				c.ioCommand <- ioOutput
				c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(*turn)
				sendWorld(c.ioOutput, world)
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

	world := receiveWorld(c.ioInput, p, c.events)

	//controls access to turn
	var turnLock = &sync.Mutex{}
	turn := 0
	go updateAliveCellsCount(world, &turn, turnLock, c.events)
	go waitKeypress(&turn, world, turnLock, c, p)

	for turn < p.Turns {
		turnLock.Lock()
		world = newWorld(turn, world, p, c.events)

		c.events <- TurnComplete{turn}
		turn++
		turnLock.Unlock()
	}

	c.ioCommand <- ioOutput
	c.ioFilename <- filenameOutput(p)

	sendWorld(c.ioOutput, world)
	c.events <- FinalTurnComplete{turn, aliveCells(world)}

	//deadlock occurs without this line
	turnLock.Lock()
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
