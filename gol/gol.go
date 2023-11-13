package gol

// Params provides the details of how to run the Game of Life and which image to load.
type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

// Run starts the processing of Game of Life. It should initialise channels and goroutines.
func Run(p Params, events chan<- Event, keyPresses <-chan rune) {

	ioCommand := make(chan ioCommand)
	ioIdle := make(chan bool)
	iofilename := make(chan string)
	iooutput := make(chan uint8)
	ioinput := make(chan uint8)
	ioChannels := ioChannels{
		command:  ioCommand,
		idle:     ioIdle,
		filename: iofilename,
		output:   iooutput,
		input:    ioinput,
	}
	go startIo(p, ioChannels)

	distributorChannels := distributorChannels{
		events:     events,
		ioCommand:  ioCommand,
		ioIdle:     ioIdle,
		ioFilename: iofilename,
		ioOutput:   iooutput,
		ioInput:    ioinput,
	}
	distributor(p, distributorChannels)
}
