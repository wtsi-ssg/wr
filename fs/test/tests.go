/*******************************************************************************
 * Copyright (c) 2021 Genome Research Ltd.
 *
 * Author: Sendu Bala <sb10@sanger.ac.uk>
 *
 * Permission is hereby granted, free of charge, to any person obtaining
 * a copy of this software and associated documentation files (the
 * "Software"), to deal in the Software without restriction, including
 * without limitation the rights to use, copy, modify, merge, publish,
 * distribute, sublicense, and/or sell copies of the Software, and to
 * permit persons to whom the Software is furnished to do so, subject to
 * the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 * EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 * MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 * IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 * CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 * TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 * SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 ******************************************************************************/

package test

// this file implements utility routines related to testing.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// readAndRestoreError records an error for restoring the *os.file handle.
type readAndRestoreError struct{}

// Error returns an error when restoring an already closed handle.
func (r *readAndRestoreError) Error() string {
	return "ReadAndRestore from closed MockStdInErr"
}

// MockStdInErr represents a mock implementation of stdin and stderr.
type MockStdInErr struct {
	origStdin    *os.File
	stdinWriter  *os.File
	origStderr   *os.File
	stderrReader *os.File
	outCh        chan []byte
}

// NewMockStdInErr mocks the stdin and captures the stderr. Between creating a
// new MockStdInErr and calling restore on it, it reads the os.Stdin and gets
// the contents of stdinText passed to NewMockStdInErr. Output to os.Stderr will
// be captured and returned from ReadAndRestoreStderr.
func NewMockStdInErr(stdinText string) (*MockStdInErr, error) {
	origStdin, stdinWriter, err := mockStdinRW(stdinText)
	if err != nil {
		return nil, err
	}

	origStderr, stderrReader, outCh, err := mockStderrRW()
	if err != nil {
		return nil, err
	}

	return &MockStdInErr{
		origStdin:    origStdin,
		stdinWriter:  stdinWriter,
		origStderr:   origStderr,
		stderrReader: stderrReader,
		outCh:        outCh,
	}, nil
}

// ReadAndRestore collects all captured stderr and returns it; it also restores
// os.Stderr to its original value.
func (se *MockStdInErr) ReadAndRestoreStderr() (string, error) {
	if se.stderrReader == nil {
		return "", &readAndRestoreError{}
	}

	os.Stderr.Close()

	out := <-se.outCh

	se.RestoreStderr()

	return string(out), nil
}

// RestoreStderr restores the stderr to its original value.
func (se *MockStdInErr) RestoreStderr() {
	os.Stderr = se.origStderr

	if se.stderrReader != nil {
		se.stderrReader.Close()
		se.stderrReader = nil
	}
}

// RestoreStdin restores the stdin to its original value.
func (se *MockStdInErr) RestoreStdin() {
	os.Stdin = se.origStdin

	if se.stdinWriter != nil {
		se.stdinWriter.Close()
		se.stdinWriter = nil
	}
}

// mockStdinRW reads os.Stdin and gets the contents of stdinText passed to it.
func mockStdinRW(stdinText string) (*os.File, *os.File, error) {
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	origStdin := os.Stdin
	os.Stdin = stdinReader

	_, err = stdinWriter.WriteString(stdinText + "\n")
	if err != nil {
		stdinWriter.Close()

		os.Stdin = origStdin

		return nil, nil, err
	}

	return origStdin, stdinWriter, nil
}

// mockStderrRW mocks the stderr and also returns the content of it.
func mockStderrRW() (*os.File, *os.File, chan []byte, error) {
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	origStderr := os.Stderr
	os.Stderr = stderrWriter

	outCh := make(chan []byte)

	go func() {
		var b bytes.Buffer
		if _, err := io.Copy(&b, stderrReader); err != nil {
			fmt.Println(err)
		}

		bytes := b.Bytes()
		outCh <- bytes
	}()

	return origStderr, stderrReader, outCh, nil
}

// FilePathInTempDir creates a new temporary directory and returns the
// absolute path to a file called basename in that directory (without
// actually creating the file).
func FilePathInTempDir(t *testing.T, basename string) string {
	tmpdir := t.TempDir()

	return filepath.Join(tmpdir, basename)
}
