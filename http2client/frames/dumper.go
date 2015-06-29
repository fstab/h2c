package frames

import (
	"fmt"
	"github.com/fatih/color"
)

var (
	prefixColor    = color.New()
	frameTypeColor = color.New(color.FgCyan)
	streamIdColor  = color.New(color.FgCyan)
	flagColor      = color.New(color.FgGreen)
	keyColor       = color.New(color.FgBlue)
	valueColor     = color.New()
)

func DumpIncoming(frame Frame) {
	dump("<-", frame)
}

func DumpOutgoing(frame Frame) {
	dump("->", frame)
}

func dump(prefix string, frame Frame) {
	prefixColor.Printf("%v ", prefix)
	switch f := frame.(type) {
	case *HeadersFrame:
		frameTypeColor.Printf("HEADERS")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpEndStream(f.EndStream)
		dumpEndHeaders(f.EndHeaders)
		if len(f.Headers) == 0 {
			keyColor.Printf("    {empty}\n")
		} else {
			for _, header := range f.Headers {
				keyColor.Printf("    %v:", header.Name)
				valueColor.Printf(" %v\n", header.Value)
			}
		}
	case *DataFrame:
		frameTypeColor.Printf("DATA")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpEndStream(f.EndStream)
		keyColor.Printf("    {%v bytes}\n", len(f.Data))
	case *SettingsFrame:
		frameTypeColor.Printf("SETTINGS")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpAck(f.Ack)
		if len(f.Settings) == 0 {
			keyColor.Printf("    {empty}\n")
		} else {
			for setting, value := range f.Settings {
				keyColor.Printf("    %v:", setting)
				valueColor.Printf(" %v\n", value)
			}
		}
	default:
		fmt.Printf("UNKNOWN (NOT IMPLEMENTED) FRAME TYPE %v", frame.Type())
	}
	fmt.Println()
}

func dumpFlag(name string, isSet bool) {
	if isSet {
		flagColor.Printf("    + %v\n", name)
	} else {
		flagColor.Printf("    - %v\n", name)
	}
}
func dumpEndStream(isSet bool) {
	dumpFlag("END_STREAM", isSet)
}

func dumpEndHeaders(isSet bool) {
	dumpFlag("END_HEADERS", isSet)
}

func dumpAck(isSet bool) {
	dumpFlag("ACK", isSet)
}
