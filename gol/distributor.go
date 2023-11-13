package gol

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
	count := 0
	for xi := -1; xi < 2; xi++ {
		xi2 := x + xi
		edgereset(xi2, params.ImageWidth)
		for yi := -1; yi < 2; yi++ {
			yi2 := yi + y
			edgereset(yi2, params.ImageHeight)
			if grid[xi2][yi2] == 255 {
				count += 1
			}
		}
	}
	return count
}

//if out of array loops the value back around again
func edgereset(i int, max int) int {
	switch i {
	case -1:
		return max - 1
	case max:
		return 0
	}
	return i
}

//cell value should return the value of a cell given its count
//count should be how many living cells are in a 3x3 block of cells centred at the cell in question
//count should already account for whether the centre cell is dead or alive
func cellValue(count int) byte {
	switch count {
	case 3:
		return 255
	case 4:
		return 255
	}
	return 0
}

// distributor divides the work between workers and interacts with other goroutines.

func distributor(p Params, c distributorChannels) {

	var world [][]byte
	var newworld [][]byte
	turn := 0

	// TODO: Execute all turns of the Game of Life.

	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
