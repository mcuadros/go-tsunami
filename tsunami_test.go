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

	trackNum := 1

	ts.TrackGain(trackNum, -70)                     // muted
	ts.TrackPlaySolo(trackNum, 0, false)            // track = 19 (aka "19.WAV"), output = 0 (aka "1L")
	ts.TrackFade(trackNum, 0, time.Second*5, false) // track 19, fade to gain of 0,
	// fade time = 5000ms, stopFlag is false = do not stop

	fmt.Println("Fading IN track 19 right now...")
	time.Sleep(time.Second * 5)

	fmt.Println("Gain set to unity (0)! Playing for 5 seconds...")
	time.Sleep(time.Second * 5)

	ts.TrackFade(trackNum, -70, time.Second*5, true) // track 3, fade to gain of -70,
	// fade time = 5000ms, stopFlag is true = stop track when fade is done

	fmt.Println("Fading OUT track 19 right now...")
	time.Sleep(time.Second * 5)

	fmt.Println("Track 19 stopped.")
	// Output:
	// Fading IN track 19 right now...
	// Gain set to unity (0)! Playing for 5 seconds...
	// Fading OUT track 19 right now...
	// Track 19 stopped.
}
