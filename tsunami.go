// Sparkfun Tsunami serial control library for golang
//
// Tsunami is a polyphonic Wav file player with 4 stereo (or 8 mono) outputs.
// Wav files can be triggered using the 16 onboard contacts, via MIDI, serial
// connection or Qwiic to a PC or other microcontroller.
package tsunami

import (
	"fmt"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// Tsunami serial connection.
type Tsunami struct {
	port *serial.Port

	voiceTable  []uint16
	version     []byte
	versionRcvd bool
	numVoices   uint8
	numTracks   uint16
	sysinfoRcvd bool
}

// NewTsunami returns a new Tsuanmi connection to the given port.
func NewTsunami(portName string) (*Tsunami, error) {
	c := &serial.Config{Name: portName, Baud: 57600,
		ReadTimeout: time.Millisecond * 5,
	}

	port, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}

	return &Tsunami{
		port:       port,
		voiceTable: make([]uint16, MAX_NUM_VOICES),
		version:    make([]byte, VERSION_STRING_LEN),
	}, nil
}

// Start initialize the serial communications.
func (t *Tsunami) Start() error {
	var txbuf = make([]byte, 5)

	// Request version string
	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x05
	txbuf[3] = CMD_GET_VERSION
	txbuf[4] = EOM

	if _, err := t.port.Write(txbuf); err != nil {
		return err
	}

	// Request system info
	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x05
	txbuf[3] = CMD_GET_SYS_INFO
	txbuf[4] = EOM

	if _, err := t.port.Write(txbuf); err != nil {
		return err
	}

	return nil
}

// IsTrackPlaying if reporting has been enabled, this function can be used to
// determine if a particular track is currently playing.
func (t *Tsunami) IsTrackPlaying(trk int) bool {
	t.update()
	for i := 0; i < MAX_NUM_VOICES; i++ {
		if t.voiceTable[i] == uint16(trk) {
			return true
		}

	}

	return false
}

// MasterGain this function immediately sets the gain of the specific stereo
// output to the specified value. The range for gain is -70 to +4. If audio is
// playing, you will hear the result immediately. If audio is not playing, the
// new gain will be used the next time a track is started.
func (t *Tsunami) MasterGain(out, gain int) error {
	var txbuf = make([]byte, 8)

	vol := uint16(gain)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x08
	txbuf[3] = CMD_MASTER_VOLUME
	txbuf[4] = byte(out & 0x07)
	txbuf[5] = byte(vol)
	txbuf[6] = byte(vol >> 8)
	txbuf[7] = EOM

	return t.write(txbuf)
}

// SetReporting this function enables or disables track reporting. When enabled,
// the Tsunami will send a message whenever a track starts or ends, specifying
// the track number. Provided you call update() periodically, the library will
// use these messages to maintain status of all tracks, allowing you to query
// if particular tracks are playing or not.
func (t *Tsunami) SetReporting(enable bool) error {
	var txbuf = make([]byte, 6)

	var e byte
	if enable {
		e = 1
	}

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x06
	txbuf[3] = CMD_SET_REPORTING
	txbuf[4] = e
	txbuf[5] = EOM

	return t.write(txbuf)
}

// GetVersion this function will return the Tsunami version string.
// This function requires bi-directional communication with Tsunami.
func (t *Tsunami) GetVersion() string {
	t.update()
	if !t.versionRcvd {
		return ""
	}

	return strings.TrimSpace(string(t.version))
}

// GetNumTracks this function will return the Tsunami version.
// This function requires bi-directional communication with Tsunami.
func (t *Tsunami) GetNumTracks() int {
	t.update()
	return int(t.numTracks)
}

// TrackPlaySolo this function stops any and all tracks that are currently
// playing and starts track number trk from the beginning. The track is routed
// to the specified stereo output. If lock is true, the track will not be
// subject to Tsunami's voice stealing algorithm.
func (t *Tsunami) TrackPlaySolo(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_PLAY_SOLO, out, flags)
}

// TrackPlayPoly this function starts track number trk from the beginning,
// blending it with any other tracks that are currently playing, including
// potentially another copy of the same track. The track is routed to the
// specified stereo output. If lock is true, the track will not be subject to
// Tsunami's voice stealing algorithm.
func (t *Tsunami) TrackPlayPoly(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_PLAY_POLY, out, flags)
}

// TrackLoad this function loads track number trk and pauses it at the beginning
// of the track. Loading muiltiple tracks and then un-pausing the all with
// resumeAllInSync() function below allows for starting multiple tracks in
// sample sync. The track is routed to the specified stereo output. If lock is
// true, the track will not be subject to Tsunami's voice stealing algorithm.
func (t *Tsunami) TrackLoad(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_LOAD, out, flags)
}

// TrackStop this function stops track number trk if it's currently playing.
// If track t is not playing, this function does nothing. No other tracks are
// affected.
func (t *Tsunami) TrackStop(trk int) error {
	return t.trackControl(trk, TRK_STOP, 0, 0)
}

// TrackPause this function pauses track number trk if it's currently playing.
// If track is not playing, this function does nothing. Keep in mind that a
// paused track is still using one of the 8 voice slots. A voice allocated to
// playing a track becomes free only when that sound is stopped or the track
// reaches the end of the file (and is not looping).
func (t *Tsunami) TrackPause(trk int) error {
	return t.trackControl(trk, TRK_PAUSE, 0, 0)
}

// TrackResume this function resumes track number trk if it's currently paused.
// If track number t is not paused, this function does nothing.
func (t *Tsunami) TrackResume(trk int) error {
	return t.trackControl(trk, TRK_RESUME, 0, 0)
}

// TrackLoop this function enables (true) or disables (false) the loop flag for
// track trk. This command does not actually start a track, only determines how
// it behaves once it is playing and reaches the end. If the loop flag is set,
// that track will loop continuously until it's stopped, in which case it will
// stop immediately but the loop flag will remain set, or until the loop flag
// is cleared, in which case it will stop when it reaches the end of the track.
// This command may be used either before a track is started or while it's playing.
func (t *Tsunami) TrackLoop(trk int, enable bool) error {
	if enable {
		return t.trackControl(trk, TRK_LOOP_ON, 0, 0)
	}

	return t.trackControl(trk, TRK_LOOP_OFF, 0, 0)
}

func (t *Tsunami) trackControl(trk, code, out, flags int) error {
	var txbuf = make([]byte, 10)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x0a
	txbuf[3] = CMD_TRACK_CONTROL
	txbuf[4] = byte(code)
	txbuf[5] = byte(trk)
	txbuf[6] = byte(trk >> 8)
	txbuf[7] = byte(out & 0x07)
	txbuf[8] = byte(flags)
	txbuf[9] = EOM

	return t.write(txbuf)
}

// StopAllTracks this commands stops any and all tracks that are currently playing.
func (t *Tsunami) StopAllTracks() error {
	var txbuf = make([]byte, 5)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x05
	txbuf[3] = CMD_STOP_ALL
	txbuf[4] = EOM

	return t.write(txbuf)
}

// ResumeAllInSync this command resumes all paused tracks within the same audio
// buffer. Any tracks that were loaded using the TrackLoad() function will
// start and remain sample locked (in sample sync) with one another.
func (t *Tsunami) ResumeAllInSync() error {
	var txbuf = make([]byte, 5)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x05
	txbuf[3] = CMD_RESUME_ALL_SYNC
	txbuf[4] = EOM

	return t.write(txbuf)
}

// TrackGain this function immediately sets the gain of track trk to the
// specified value. The range for gain is -70 to +10. A value of 0 (no gain)
// plays the track at the nominal value in the wav file. This is the default
// gain for every track until changed. A value of -70 is completely muted. If
// the track is playing, you will hear the result immediately. If the track is
// not playing, the gain will be used the next time the track is started.
// Every track can have its own gain.
//
// Because the effect is immediate, large changes can produce ubrupt results.
// If you want to fade in or fade out a track, send small changes spaced out at
// regular intervals. Increment or decrementing by 1 every 20 to 50 msecs
// produces nice smooth fades. Better yet, use the trackFade() function below.
func (t *Tsunami) TrackGain(trk, gain int) error {
	var txbuf = make([]byte, 9)

	vol := uint16(gain)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x09
	txbuf[3] = CMD_TRACK_VOLUME
	txbuf[4] = byte(trk)
	txbuf[5] = byte(trk >> 8)
	txbuf[6] = byte(vol)
	txbuf[7] = byte(vol >> 8)
	txbuf[8] = EOM

	return t.write(txbuf)
}

// TrackFade this command initiates a hardware volume fade on track number trk
// if it is currently playing. The track volume will transition smoothly from
// the current value to the target gain in the specified number of milliseconds.
// If the stopFlag is non-zero, the track will be stopped at the completion of
// the fade (for fade-outs.)
func (t *Tsunami) TrackFade(trk, gain int, d time.Duration, stopFlag bool) error {
	var txbuf = make([]byte, 12)
	vol := uint16(gain)

	stop := 0
	if stopFlag {
		stop = 1
	}

	time := d.Milliseconds()

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x0c
	txbuf[3] = CMD_TRACK_FADE
	txbuf[4] = byte(trk)
	txbuf[5] = byte(trk >> 8)
	txbuf[6] = byte(vol)
	txbuf[7] = byte(vol >> 8)
	txbuf[8] = byte(time)
	txbuf[9] = byte(time >> 8)
	txbuf[10] = byte(stop)
	txbuf[11] = EOM

	return t.write(txbuf)
}

// SamplerateOffset this function immediately sets sample-rate offset, or
// playback speed / pitch, of the specified stereo output. The range for for
// the offset is -32767 to +32676, giving a speed range of 1/2x to 2x, or a
// pitch range of down one octave to up one octave. If audio is playing, you
// will hear the result immediately. If audio is not playing, the new
// sample-rate offset will be used the next time a track is started.
func (t *Tsunami) SamplerateOffset(out, offset int) error {
	var txbuf = make([]byte, 8)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x08
	txbuf[3] = CMD_SAMPLERATE_OFFSET
	txbuf[4] = byte(0)

	off := uint16(offset)
	txbuf[5] = byte(off)
	txbuf[6] = byte(off >> 8)
	txbuf[7] = EOM

	return t.write(txbuf)
}

// SetTriggerBank this function sets the trigger bank. The bank range is 1 - 32.
// Each bank will offset the normal trigger function track assignment by 16.
// For bank 1, the default, trigger one maps to track 1. For bank 2, trigger 1
// maps to track 17, trigger 2 to track 18, and so on.
func (t *Tsunami) SetTriggerBank(bank int) error {
	var txbuf = make([]byte, 6)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x06
	txbuf[3] = CMD_SET_TRIGGER_BANK
	txbuf[4] = byte(bank)
	txbuf[5] = EOM

	return t.write(txbuf)
}

// SetInputMix this function controls the routing of the audio input channels.
// For bits 1 through 4, a "1" causes the 2 input channels to be mixed into the
// corresponding output pair. As an example, to route the audio input to output
// pairs 1, 2 and 4, the syntax is:  SetInputMix(IMIX_OUT1 | IMIX_OUT2 | IMIX_OUT4)
//
// The routing is immediate and does no ramping, so to avoid pops, be sure that
// the input is quiet when switching.
func (t *Tsunami) SetInputMix(mix int) error {
	var txbuf = make([]byte, 6)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x06
	txbuf[3] = CMD_SET_INPUT_MIX
	txbuf[4] = byte(mix)
	txbuf[5] = EOM

	return t.write(txbuf)
}

// SetMidiBank this function sets the MIDI bank. The bank range is 1 - 32. Each
// bank will offset the MIDI Note number to track assignment by 128. For
// bank 1, the default, MIDI Note number maps to track 1. For bank 2, MIDI Note
// number 1 maps to track 129, MIDI Note number 2 to track 130, and so on.
func (t *Tsunami) SetMidiBank(bank int) error {
	var txbuf = make([]byte, 6)

	txbuf[0] = SOM1
	txbuf[1] = SOM2
	txbuf[2] = 0x06
	txbuf[3] = CMD_SET_MIDI_BANK
	txbuf[4] = byte(bank)
	txbuf[5] = EOM

	return t.write(txbuf)
}

func (t *Tsunami) write(b []byte) error {
	n, err := t.port.Write(b)
	if err != nil {
		return err
	}

	if n != 10 {
		return fmt.Errorf("unexpected bytes written %d", n)
	}

	return nil
}

func (t *Tsunami) update() error {
	rxMessage := make([]uint8, MAX_MESSAGE_LEN)
	var rxCount byte
	var rxLen byte
	var rxMsgReady bool

	txbuf := make([]byte, 50)

	for {
		n, _ := t.port.Read(txbuf)
		if n == 0 {
			break
		}

		for _, dat := range txbuf[:n] {
			if (rxCount == 0) && (dat == SOM1) {
				rxCount++
			} else if rxCount == 1 {
				if dat == SOM2 {
					rxCount++
				} else {
					rxCount = 0
					return fmt.Errorf("bad msg 1")
				}
			} else if rxCount == 2 {
				if dat <= MAX_MESSAGE_LEN {
					rxCount++
					rxLen = dat - 1
				} else {
					rxCount = 0
					return fmt.Errorf("bad msg 2")
				}
			} else if (rxCount > 2) && (rxCount < rxLen) {
				rxMessage[rxCount-3] = dat
				rxCount++
			} else if rxCount == rxLen {
				if dat == EOM {
					rxMsgReady = true
				} else {
					rxCount = 0
					return fmt.Errorf("bad msg 3")
				}
			} else {
				rxCount = 0
				return fmt.Errorf("bad msg 4")
			}

			if rxMsgReady {
				switch rxMessage[0] {

				case RSP_TRACK_REPORT:
					track := uint16(rxMessage[2])
					track = (track << 8) + uint16(rxMessage[1]) + 1
					voice := rxMessage[3]
					if voice < MAX_NUM_VOICES {
						if rxMessage[4] == 0 {
							if track == t.voiceTable[voice] {
								t.voiceTable[voice] = 0xffff
							}
						} else {
							t.voiceTable[voice] = track
						}
					}
					// ==========================
					//fmt.Printf("Track %d", track)
					//if rxMessage[4] == 0 {
					//	fmt.Println(" off")
					//} else {
					//	fmt.Println(" on")
					//}
					// ==========================

				case RSP_VERSION_STRING:
					for i := 0; i < (VERSION_STRING_LEN - 1); i++ {
						t.version[i] = rxMessage[i+1]
					}

					t.version[VERSION_STRING_LEN-1] = 0
					t.versionRcvd = true

					// ==========================
					//fmt.Println(string(t.version), t.versionRcvd)
					// ==========================

				case RSP_SYSTEM_INFO:
					t.numVoices = byte(rxMessage[1])
					t.numTracks = uint16(rxMessage[3])
					t.numTracks = (t.numTracks << 8) + uint16(rxMessage[2])
					t.sysinfoRcvd = true

					// ==========================
					//fmt.Println("sysinfoRcvd", t.numVoices, t.numTracks)
					// ==========================
				}

				rxCount = 0
				rxLen = 0
				rxMsgReady = false

			} // if (rxMsgReady)
		} // while (TsunamiSerial.available() > 0)
	}

	return nil
}

// Close should be called to close the connection with the port.
func (t *Tsunami) Close() error {
	return t.port.Close()
}

const (
	CMD_GET_VERSION       = 1
	CMD_GET_SYS_INFO      = 2
	CMD_TRACK_CONTROL     = 3
	CMD_STOP_ALL          = 4
	CMD_MASTER_VOLUME     = 5
	CMD_TRACK_VOLUME      = 8
	CMD_TRACK_FADE        = 10
	CMD_RESUME_ALL_SYNC   = 11
	CMD_SAMPLERATE_OFFSET = 12
	CMD_SET_REPORTING     = 13
	CMD_SET_TRIGGER_BANK  = 14
	CMD_SET_INPUT_MIX     = 15
	CMD_SET_MIDI_BANK     = 16

	TRK_PLAY_SOLO      = 0
	TRK_PLAY_POLY      = 1
	TRK_PAUSE          = 2
	TRK_RESUME         = 3
	TRK_STOP           = 4
	TRK_LOOP_ON        = 5
	TRK_LOOP_OFF       = 6
	TRK_LOAD           = 7
	RSP_VERSION_STRING = 129
	RSP_SYSTEM_INFO    = 130
	RSP_STATUS         = 131
	RSP_TRACK_REPORT   = 132

	MAX_MESSAGE_LEN    = 32
	MAX_NUM_VOICES     = 18
	VERSION_STRING_LEN = 23

	SOM1 = 0xf0
	SOM2 = 0xaa
	EOM  = 0x55

	IMIX_OUT1 = 0x01
	IMIX_OUT2 = 0x02
	IMIX_OUT3 = 0x04
	IMIX_OUT4 = 0x08
)
