package codecs

import (
	"github.com/driskell/log-courier/src/lc-lib/config"
	"sync"
	"testing"
	"time"
)

var multilineTest *testing.T
var multilineLines int
var multilineLock sync.Mutex

func createMultilineCodec(unused map[string]interface{}, callback CallbackFunc, t *testing.T) Codec {
	config := config.NewConfig()
	config.General.MaxLineBytes = 1048576
	config.General.SpoolMaxBytes = 10485760

	factory, err := NewMultilineCodecFactory(config, "", unused, "multiline")
	if err != nil {
		t.Logf("Failed to create multiline codec: %s", err)
		t.FailNow()
	}

	return NewCodec(factory, callback, 0)
}

func checkMultiline(startOffset int64, endOffset int64, text string) {
	multilineLock.Lock()
	defer multilineLock.Unlock()
	multilineLines++

	if multilineLines == 1 {
		if text != "DEBUG First line\nNEXT line\nANOTHER line" {
			multilineTest.Logf("Event data incorrect [% X]", text)
			multilineTest.FailNow()
		}

		if startOffset != 0 {
			multilineTest.Logf("Event start offset is incorrect [%d]", startOffset)
			multilineTest.FailNow()
		}

		if endOffset != 5 {
			multilineTest.Logf("Event end offset is incorrect [%d]", endOffset)
			multilineTest.FailNow()
		}

		return
	}

	if text != "DEBUG Next line" {
		multilineTest.Logf("Event data incorrect [% X]", text)
		multilineTest.FailNow()
	}

	if startOffset != 6 {
		multilineTest.Logf("Event start offset is incorrect [%d]", startOffset)
		multilineTest.FailNow()
	}

	if endOffset != 7 {
		multilineTest.Logf("Event end offset is incorrect [%d]", endOffset)
		multilineTest.FailNow()
	}
}

func TestMultilinePrevious(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"pattern": "^(ANOTHER|NEXT) ",
		"what":    "previous",
		"negate":  false,
	}, checkMultiline, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if multilineLines != 1 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilinePreviousNegate(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"pattern": "^DEBUG ",
		"what":    "previous",
		"negate":  true,
	}, checkMultiline, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if multilineLines != 1 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilinePreviousTimeout(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"pattern":          "^(ANOTHER|NEXT) ",
		"what":             "previous",
		"negate":           false,
		"previous timeout": "3s",
	}, checkMultiline, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	// Allow a second
	time.Sleep(time.Second)

	multilineLock.Lock()
	if multilineLines != 1 {
		t.Logf("Timeout triggered too early")
		t.FailNow()
	}
	multilineLock.Unlock()

	// Allow 5 seconds
	time.Sleep(5 * time.Second)

	multilineLock.Lock()
	if multilineLines != 2 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}
	multilineLock.Unlock()

	offset := codec.Teardown()
	if offset != 7 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineNext(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"pattern": "^(DEBUG|NEXT) ",
		"what":    "next",
		"negate":  false,
	}, checkMultiline, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if multilineLines != 1 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func TestMultilineNextNegate(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"pattern": "^ANOTHER ",
		"what":    "next",
		"negate":  true,
	}, checkMultiline, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "NEXT line")
	codec.Event(4, 5, "ANOTHER line")
	codec.Event(6, 7, "DEBUG Next line")

	if multilineLines != 1 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}

func checkMultilineMaxBytes(startOffset int64, endOffset int64, text string) {
	multilineLines++

	if multilineLines == 1 {
		if text != "DEBUG First line\nsecond line\nthi" {
			multilineTest.Logf("Event data incorrect [% X]", text)
			multilineTest.FailNow()
		}

		return
	}

	if text != "rd line" {
		multilineTest.Logf("Second event data incorrect [% X]", text)
		multilineTest.FailNow()
	}
}

func TestMultilineMaxBytes(t *testing.T) {
	multilineTest = t
	multilineLines = 0

	codec := createMultilineCodec(map[string]interface{}{
		"max multiline bytes": int64(32),
		"pattern":             "^DEBUG ",
		"negate":              true,
	}, checkMultilineMaxBytes, t)

	// Send some data
	codec.Event(0, 1, "DEBUG First line")
	codec.Event(2, 3, "second line")
	codec.Event(4, 5, "third line")
	codec.Event(6, 7, "DEBUG Next line")

	if multilineLines != 2 {
		t.Logf("Wrong line count received")
		t.FailNow()
	}

	offset := codec.Teardown()
	if offset != 5 {
		t.Error("Teardown returned incorrect offset: ", offset)
	}
}
