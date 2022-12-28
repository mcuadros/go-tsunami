go-tsunami [![GoDoc](http://godoc.org/github.com/mcuadros/go-tsunami?status.svg)](http://godoc.org/github.com/mcuadros/go-tsunami)
==============================

Sparkfun Tsunami serial control library for golang 

[Tsunami](https://www.sparkfun.com/products/18159) is a polyphonic Wav file 
player with 4 stereo (or 8 mono) outputs. Wav files can be triggered using the
16 onboard contacts, via MIDI, serial connection or Qwiic to a PC or other
microcontroller.

This library is a manual transpilation from the official library 
[Tsunami-Arduino-Serial-Library](https://github.com/robertsonics/Tsunami-Arduino-Serial-Library).

Installation
------------

The recommended way to install go-tsunami

```
go get github.com/mcuadros/go-tsunami
```

Example
-------

```go
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
```


License
-------

MIT, see [LICENSE](LICENSE)
