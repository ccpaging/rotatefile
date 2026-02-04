// Copyright (C) 2017, ccpaging <ccpaging@gmail.com>.  All rights reserved.

package rotatefile

import (
	"io/ioutil"
	"os"
	"runtime"
	"testing"
)

var testFiles []string = []string{"_test.log", "_test.1.log"}
var testString = "hello, world"
var testLongString = "Everything is created now (notice that I will be printing to the file)"
var benchLogFiles []string = []string{"_benchlog.log", "_benchlog.1.log"}

func TestWriter(t *testing.T) {
	testFile := testFiles[0]

	f, _ := Open(testFile)
	defer os.Remove(testFile)

	f.Write([]byte(testString))
	f.Close()

	runtime.Gosched()

	if contents, err := ioutil.ReadFile(testFile); err != nil {
		t.Errorf("read(%q): %s", testFiles, err)
	} else if len(contents) != 12 {
		t.Errorf("malformed file: %q (%d bytes)", string(contents), len(contents))
	}
}

func TestRolling(t *testing.T) {
	f, _ := OpenFile(testFiles[0], 5*1024, 1)

	for j := 0; j < 15; j++ {
		for i := 0; i < 200/(j+1); i++ {
			f.Write([]byte(testLongString + "\n"))
		}
	}

	f.Close()

	runtime.Gosched()

	for _, testFile := range testFiles {
		if contents, err := ioutil.ReadFile(testFile); err != nil {
			t.Errorf("read(%q): %s", testFile, err)
		} else if len(contents) != 213 && len(contents) != 5183 {
			t.Errorf("malformed file: %q (%d bytes)", string(contents), len(contents))
		}

		os.Remove(testFile)
	}
}

func BenchmarkNoBuffer(b *testing.B) {
	f, _ := Open(benchLogFiles[0])
	f.Buffersize = 0
	defer func() {
		f.Close()
		os.Remove(benchLogFiles[0])
		os.Remove(benchLogFiles[1])
	}()
	b.StopTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		f.Write([]byte(testLongString + "\n"))
	}
	b.StopTimer()
}

func BenchmarkBuffered(b *testing.B) {
	f, _ := Open(benchLogFiles[0])
	defer func() {
		f.Close()
		os.Remove(benchLogFiles[0])
		os.Remove(benchLogFiles[1])
	}()
	b.StopTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		f.Write([]byte(testLongString + "\n"))
	}
	b.StopTimer()
}
