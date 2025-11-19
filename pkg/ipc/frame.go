package ipc

import (
	"encoding/binary"
	"io"
)

// ReadFrame reads a length-prefixed payload from r.
func ReadFrame(r io.Reader) ([]byte, error) {
	return readFrame(r)
}

// WriteFrame writes payload to w with a 4-byte little-endian length prefix.
func WriteFrame(w io.Writer, payload []byte) error {
	return writeFrame(w, payload)
}

func readFrame(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, payload []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
