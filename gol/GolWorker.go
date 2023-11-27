package gol

import (
	"sync"
)

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
func stageConverter(startY, endY, startX, endX int, world, newWorld [][]byte, params Params) {
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, params), world[y][x])
			if newWorld[y][x] != world[y][x] {
			}
		}
	}
}
func partWorld(turn, startY, endY, startX, endX int, world, newWorld [][]byte, params Params, group *sync.WaitGroup) {
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, params), world[y][x])
		}
	}
	group.Done()
}
func newworld(turn int, world, newWorld [][]byte, p Params, mutex *sync.Mutex) {
	mutex.Lock()
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
			go partWorld(turn, startY, endY, 0, p.ImageWidth, world, newWorld, p, &wg)
		}
		wg.Wait()
	}
	mutex.Unlock()
}
func allTurns(p Params, worldEven, worldOdd [][]byte) {

	//controls access to the odd or even matrix
	//var oddMutex = &sync.Mutex{}
	//var EvenMutex = &sync.Mutex{}
	//controls access to turn
	var turnLock = &sync.Mutex{}
	turn := 0
	for turn < p.Turns {
		turnLock.Lock()
		if turn%2 == 0 {
			stageConverter(0, p.ImageHeight, 0, p.ImageWidth, worldEven, worldOdd, p)
		} else {
			stageConverter(0, p.ImageHeight, 0, p.ImageWidth, worldOdd, worldEven, p)
		}
		turn++
		turnLock.Unlock()
	}
	//deadlock occurs without this line
	turnLock.Lock()
}
