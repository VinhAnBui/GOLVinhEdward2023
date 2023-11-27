package Server

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

//GOL logic
//count3x3 counts how many live cells there are in a 3 x 3 area centered around some x y
func count3x3(grid [][]byte, x, y, height, width int) int {
	//fmt.Println(x, y)
	count := 0
	for xi := -1; xi < 2; xi++ {
		xi2 := x + xi
		//fmt.Println("xi:", xi)
		xi2 = edgereset(xi2, width)
		//fmt.Println("xi2:", xi2)
		for yi := -1; yi < 2; yi++ {
			//fmt.Println("yi:", yi)
			yi2 := yi + y
			yi2 = edgereset(yi2, height)
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
func stageConverter(startY, endY, startX, endX, height, width int, world, newWorld [][]byte) {
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			newWorld[y][x] = cellValue(count3x3(world, x, y, height, width), world[y][x])
		}
	}
}

type allTurns struct{}

func (t *allTurns) allTurns(req stubs.Request, res *stubs.Response) (err error) {
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
func main() {
	pAddr := flag.String("port", "8030", "port to liston on")
	flag.Parse()
	rpc.Register(allTurns{})
	rand.Seed(time.Now().UnixNano())
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
