package tsunami_test

import (
	"fmt"
	"time"

	"github.com/mcuadros/go-tsunami"
)

func ExampleTsunami() {
	ts, err := tsunami.NewTsunami("/dev/ttyUSB0")
	if err != nil {
		panic(err)
	}

	defer ts.Close()

	if err := ts.Start(); err != nil {
		panic(err)
	}

	trackNum := 1002

	fmt.Println(ts.GetNumTracks())

	ts.TrackGain(trackNum, 70)                      // muted
	ts.TrackPlaySolo(trackNum, 0, false)            // track = 19 (aka "19.WAV"), output = 0 (aka "1L")
	ts.TrackFade(trackNum, 0, time.Second*5, false) // track 19, fade to gain of 0,

	fmt.Println("Track 19 stopped.")
	// Output:
	// Fading IN track 19 right now...
	// Gain set to unity (0)! Playing for 5 seconds...
	// Fading OUT track 19 right now...
	// Track 19 stopped.
}
