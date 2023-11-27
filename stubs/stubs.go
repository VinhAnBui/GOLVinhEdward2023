package stubs

var AllTurns = "golWorker.allTurns"

type Response struct {
	World [][]byte
}
type Request struct {
	WorldEven   [][]byte
	ImageWidth  int
	ImageHeight int
	Turns       int
}
