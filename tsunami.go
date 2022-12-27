package tsunami

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

type Tsunami struct {
	port *serial.Port

	voiceTable  []uint16
	version     []byte
	versionRcvd bool
	numVoices   uint8
	numTracks   uint16
	sysinfoRcvd bool
}

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

func (t *Tsunami) Update() {
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
					fmt.Println("Bad msg 1")
				}
			} else if rxCount == 2 {
				if dat <= MAX_MESSAGE_LEN {
					rxCount++
					rxLen = dat - 1
				} else {
					rxCount = 0
					fmt.Println("Bad msg 2")
				}
			} else if (rxCount > 2) && (rxCount < rxLen) {
				rxMessage[rxCount-3] = dat
				rxCount++
			} else if rxCount == rxLen {
				if dat == EOM {
					rxMsgReady = true
				} else {
					rxCount = 0
					fmt.Println("Bad msg 3")
				}
			} else {
				rxCount = 0
				fmt.Println("Bad msg 4")
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
					fmt.Printf("Track %d", track)
					if rxMessage[4] == 0 {
						fmt.Println(" off")
					} else {
						fmt.Println(" on")
					}
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
}

func (t *Tsunami) IsTrackPlaying(trk int) bool {
	t.Update()
	for i := 0; i < MAX_NUM_VOICES; i++ {
		if t.voiceTable[i] == uint16(trk) {
			return true
		}

	}

	return false
}

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

	if _, err := t.port.Write(txbuf); err != nil {
		return err
	}

	return nil
}

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

	if _, err := t.port.Write(txbuf); err != nil {
		return err
	}

	return nil
}

func (t *Tsunami) GetVersion() string {
	t.Update()
	if !t.versionRcvd {
		return ""
	}

	return string(t.version)
}

func (t *Tsunami) GetNumTracks() int {
	t.Update()
	return int(t.numTracks)
}

func (t *Tsunami) TrackPlaySolo(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_PLAY_SOLO, out, flags)
}

func (t *Tsunami) TrackPlayPoly(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_PLAY_POLY, out, flags)
}

func (t *Tsunami) TrackLoad(trk, out int, lock bool) error {
	var flags = 0
	if lock {
		flags |= 0x01
	}

	return t.trackControl(trk, TRK_LOAD, out, flags)
}

func (t *Tsunami) TrackStop(trk int) error {
	return t.trackControl(trk, TRK_STOP, 0, 0)
}

func (t *Tsunami) TrackPause(trk int) error {
	return t.trackControl(trk, TRK_PAUSE, 0, 0)
}

func (t *Tsunami) TrackResume(trk int) error {
	return t.trackControl(trk, TRK_RESUME, 0, 0)
}

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
	fmt.Println(byte(trk))
	txbuf[5] = byte(trk)
	txbuf[6] = byte(trk >> 8)
	txbuf[7] = byte(out & 0x07)
	txbuf[8] = byte(flags)
	txbuf[9] = EOM

	n, err := t.port.Write(txbuf)
	if err != nil {
		return err
	}

	if n != 10 {
		return fmt.Errorf("unexpected bytes written %d", n)
	}

	return err
}

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
