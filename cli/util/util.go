package util

import "github.com/fstab/h2c/http2client/frames"

func SliceContainsFrameType(s []frames.Type, t frames.Type) bool {
	for _, e := range s {
		if e == t {
			return true
		}
	}
	return false
}
