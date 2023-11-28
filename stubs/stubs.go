package stubs

var AllTurns = "AllTurns.AllTurns"

type Response struct {
	World [][]byte
}
type Request struct {
	WorldEven   [][]byte
	ImageWidth  int
	ImageHeight int
	Turns       int
}
