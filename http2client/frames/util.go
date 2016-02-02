package frames

func FrameNameToType(name string) (Type, bool) {
	t, ok := map[string]Type{
		"DATA":          DATA_TYPE,
		"HEADERS":       HEADERS_TYPE,
		"PRIORITY":      PRIORITY_TYPE,
		"RST_STREAM":    RST_STREAM_TYPE,
		"SETTINGS":      SETTINGS_TYPE,
		"PUSH_PROMISE":  PUSH_PROMISE_TYPE,
		"PING":          PING_TYPE,
		"GOAWAY":        GOAWAY_TYPE,
		"WINDOW_UPDATE": WINDOW_UPDATE_TYPE,
		// TODO: "CONTINUATION"
	}[name]
	return t, ok
}

func AllFrameTypes() []Type {
	return []Type{
		DATA_TYPE,
		HEADERS_TYPE,
		PRIORITY_TYPE,
		RST_STREAM_TYPE,
		SETTINGS_TYPE,
		PUSH_PROMISE_TYPE,
		PING_TYPE,
		GOAWAY_TYPE,
		WINDOW_UPDATE_TYPE,
		// TODO: CONTINUATION
	}
}
