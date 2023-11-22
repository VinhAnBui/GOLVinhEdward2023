package gol

import (
	"fmt"
	"strconv"
	"sync"
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

//count3x3 counts how many live cells there are in a 3 x 3 area centered around some x y
// accounts for edges and corners
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
//max is max width or height
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
//count should be how many living cells are in a 3x3 block of cells centred at the cell in question
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

//generates the filename
func generateFile(p Params) string {
	s := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth)
	fmt.Println(s)
	return s
}

//recieves an array of bytes from ioInput
func recieveworld(ioInput <-chan uint8, p Params) [][]byte {
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
			world[y][x] = val
		}
	}
	return world
}
func sendsworld(ioOutput chan<- uint8, p Params, world [][]byte) {
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			ioOutput <- world[y][x]
		}
	}
}
func partWorld(startY, endY, startX, endX int, world, newWorld [][]byte, params Params, group *sync.WaitGroup) {
	stageConverter(startY, endY, startX, endX, world, newWorld, params)
	group.Done()
}
func stageConverter(startY, endY, startX, endX int, world, newWorld [][]byte, params Params) {

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, params), world[y][x])
		}
	}
}
func newworld(world, newWorld [][]byte, p Params) {
	if p.Threads == 1 {
		stageConverter(0, p.ImageHeight, 0, p.ImageWidth, world, newWorld, p)
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
			go partWorld(startY, endY, 0, p.ImageWidth, world, newWorld, p, &wg)
		}
		wg.Wait()
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
func alivecells(world [][]byte) []util.Cell {

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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	fmt.Println("1")
	c.ioCommand <- ioInput
	//sends the filename to io
	c.ioFilename <- generateFile(p)
	fmt.Println("2")

	worldEven := recieveworld(c.ioInput, p)

	worldOdd := make([][]byte, p.ImageHeight)
	for i := range worldOdd {
		worldOdd[i] = make([]byte, p.ImageWidth)
	}

	turn := 0
	for turn < p.Turns {
		if turn%2 == 0 {
			newworld(worldEven, worldOdd, p)
		} else {
			newworld(worldOdd, worldEven, p)
		}
		turn++
	}

	if turn%2 != 0 {
		c.events <- FinalTurnComplete{turn, alivecells(worldOdd)}
	} else {
		c.events <- FinalTurnComplete{turn, alivecells(worldEven)}
	}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
