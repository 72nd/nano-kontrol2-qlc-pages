package main

import (
	"bufio"
	"fmt"
	"github.com/hypebeast/go-osc/osc"
	"github.com/youpy/go-coremidi"
	"log"
	"os"
	"math"
	"strings"
)

const debug = false
const inputDevice = "nanoKONTROL2"
const buttonPressed = 127

type InterfaceElement int

const (
	ButtonTrackPrevious  InterfaceElement = 58
	ButtonTrackNext                       = 59
	ButtonCycle                           = 46
	ButtonMarkerSet                       = 60
	ButtonMarkerPrevious                  = 61
	ButtonMarkerNext                      = 62
	ButtonRewind                          = 43
	ButtonForward                         = 44
	ButtonStop                            = 42
	ButtonPlay                            = 41
	ButtonRecord                          = 45
	Slider1                               = 0
	Slider8                               = 7
	Knob1                                 = 16
	Knob8                                 = 23
	ButtonSolo1                           = 32
	ButtonSolo8                           = 39
	ButtonMute1                           = 48
	ButtonMute8                           = 55
	ButtonReccord1                        = 64
	ButtonReccord8                        = 71
)

const sliderDelta = 1
const knobDelta = -15
const muteDelta = -47
const soloDelta = -31
const recordDelta = -63

var group = "s1"
var groupNames = map[string]string{}
var page = 1
var client *osc.Client

func handler(src coremidi.Source, pkg coremidi.Packet) {
	element := InterfaceElement(pkg.Data[1])
	value := int(pkg.Data[2])

	viewChanged := false
	if element == ButtonTrackPrevious && value == buttonPressed && page > 1 {
		page -= 1
		viewChanged = true
	} else if element == ButtonTrackNext && value == buttonPressed && page < 8 {
		page += 1
		viewChanged = true
	} else if element >= Slider1 && element <= Slider8 {
		sendOsc("slider", element, value, sliderDelta)
	} else if element >= Knob1 && element <= Knob8 {
		sendOsc("knob", element, value, knobDelta)
	} else if element >= ButtonSolo1 && element <= ButtonSolo8 && value == buttonPressed {
		group = fmt.Sprintf("s%d", element+soloDelta)
		viewChanged = true
	} else if element >= ButtonMute1 && element <= ButtonMute8 && value == buttonPressed {
		group = fmt.Sprintf("m%d", element+muteDelta)
		viewChanged = true
	} else if element >= ButtonReccord1 && element <= ButtonReccord8 && value == buttonPressed {
		group = fmt.Sprintf("r%d", element+recordDelta)
		viewChanged = true
	} else {
		address := ""
		switch element {
		case ButtonCycle:
			address = "cycle"
		case ButtonMarkerSet:
			address = "marker/set"
		case ButtonMarkerPrevious:
			address = "marker/previous"
		case ButtonMarkerNext:
			address = "marker/next"
		case ButtonRewind:
			address = "rewind"
		case ButtonForward:
			address = "forward"
		case ButtonStop:
			address = "stop"
		case ButtonPlay:
			address = "play"
		case ButtonRecord:
			address = "record"
		}
		if address != "" {
			msg := osc.NewMessage(fmt.Sprintf("/%s", address))
			msg.Append(int32(255))
			client.Send(msg)
		}
	}

	if viewChanged {
		groupDisplay := group
		if value, ok := groupNames[group]; ok {
			groupDisplay = fmt.Sprintf("%s (%s)", group, value)
		}
		fmt.Printf("Group: %s, Page: %d\n", groupDisplay, page)
	}
	if debug {
		fmt.Printf(
			"device: %v, manufacturer: %v, soure: %v, data: %v\n",
			src.Entity().Device().Name(),
			src.Manufacturer(),
			src.Name(),
			pkg.Data,
		)
	}
}

func sendOsc(outType string, element InterfaceElement, value int, delta int) {
	channel := int(element) + delta + (page-1)*8
	msgValue := float64(value) / 127.0 * 255.0
	msg := osc.NewMessage(fmt.Sprintf("/%s/%s/%d", group, outType, channel))
	msg.Append(int32(math.Round(msgValue)))
	client.Send(msg)
}

func connectToMidi() {
	client, err := coremidi.NewClient("client")
	if err != nil {
		log.Panic(err)
	}
	port, err := coremidi.NewInputPort(client, "iput", handler)
	if err != nil {
		log.Panic(err)
	}
	sources, err := coremidi.AllSources()
	if err != nil {
		log.Panic(err)
	}
	foundInputDevice := false
	for _, source := range sources {
		func(src coremidi.Source) {
			if inputDevice == src.Entity().Device().Name() {
				port.Connect(src)
				foundInputDevice = true
			}
		}(source)
	}
	if foundInputDevice {
		fmt.Println("MIDI input connected.")
	} else {
		log.Fatalf("No input %s device found\n", inputDevice)
	}
}

func loadNames() {
	if len(os.Args) != 2 {
		return
	}
	file, err := os.Open(os.Args[1:][0])
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		values := strings.Split(scanner.Text(), "=")
		groupNames[strings.TrimSpace(values[0])] = strings.TrimSpace(values[1])
	}
}

func main() {
	loadNames()
	connectToMidi()
	client = osc.NewClient("127.0.0.1", 7700)
	ch := make(chan int)
	<-ch
}
