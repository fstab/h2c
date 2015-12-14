package daemon

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/fstab/h2c/http2client/frames"
)

var (
	prefixColor    = color.New()
	frameTypeColor = color.New(color.FgCyan)
	streamIdColor  = color.New(color.FgCyan)
	flagColor      = color.New(color.FgGreen)
	keyColor       = color.New(color.FgBlue)
	valueColor     = color.New()
)

func DumpIncoming(frame frames.Frame) {
	dump("<-", frame)
}

func DumpOutgoing(frame frames.Frame) {
	dump("->", frame)
}

func dump(prefix string, frame frames.Frame) {
	prefixColor.Printf("%v ", prefix)
	switch f := frame.(type) {
	case *frames.HeadersFrame:
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
	case *frames.DataFrame:
		frameTypeColor.Printf("DATA")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpEndStream(f.EndStream)
		keyColor.Printf("    {%v bytes}\n", len(f.Data))
	case *frames.PriorityFrame:
		frameTypeColor.Printf("PRIORITY")
		keyColor.Printf("    Stream dependency:")
		valueColor.Printf(" %v\n", f.StreamDependencyId)
		keyColor.Printf("    Weight:")
		valueColor.Printf(" %v\n", f.Weight)
		keyColor.Printf("    Exclusive:")
		valueColor.Printf(" %v\n", f.Exclusive)
	case *frames.SettingsFrame:
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
	case *frames.PushPromiseFrame:
		frameTypeColor.Printf("PUSH_PROMISE")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpEndHeaders(f.EndHeaders)
		keyColor.Printf("    Promised Stream Id:")
		valueColor.Printf(" %v\n", f.PromisedStreamId)
		if len(f.Headers) == 0 {
			keyColor.Printf("    {empty}\n")
		} else {
			for _, header := range f.Headers {
				keyColor.Printf("    %v:", header.Name)
				valueColor.Printf(" %v\n", header.Value)
			}
		}
	case *frames.RstStreamFrame:
		frameTypeColor.Printf("RST_STREAM")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		keyColor.Printf("    Error code:")
		valueColor.Printf(" %v\n", f.ErrorCode.String())
	case *frames.PingFrame:
		frameTypeColor.Printf("PING")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		dumpAck(f.Ack)
		keyColor.Printf("    payload:")
		valueColor.Printf(" 0x%016x\n", f.Payload)
	case *frames.GoAwayFrame:
		frameTypeColor.Printf("GOAWAY")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		keyColor.Printf("    Last stream id:")
		valueColor.Printf(" %v\n", f.LastStreamId)
		keyColor.Printf("    Error code:")
		valueColor.Printf(" %v\n", f.ErrorCode.String())
	case *frames.WindowUpdateFrame:
		frameTypeColor.Printf("WINDOW_UPDATE")
		streamIdColor.Printf("(%v)\n", f.StreamId)
		keyColor.Printf("    Window size increment:")
		valueColor.Printf(" %v\n", f.WindowSizeIncrement)
	default:
		frameTypeColor.Printf("UNKNOWN (NOT IMPLEMENTED) FRAME TYPE %v\n", frame.Type())
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
