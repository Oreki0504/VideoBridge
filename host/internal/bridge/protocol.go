package bridge

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

const MaxFrame = 768 * 1024

type Request struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type Event struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
type Writer struct {
	mu  sync.Mutex
	Out io.Writer
}

func ReadFrame(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.NativeEndian.Uint32(header[:])
	if size == 0 || size > MaxFrame {
		return nil, errors.New("invalid frame size")
	}
	data := make([]byte, size)
	_, err := io.ReadFull(r, data)
	return data, err
}

func (w *Writer) Send(id, kind string, payload any) error {
	data, err := json.Marshal(Event{1, id, kind, payload})
	if err != nil || len(data) > MaxFrame {
		return errors.New("invalid outbound frame")
	}
	var header [4]byte
	binary.NativeEndian.PutUint32(header[:], uint32(len(data)))
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := writeAll(w.Out, header[:]); err != nil {
		return err
	}
	return writeAll(w.Out, data)
}
func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
func Decode(data []byte, out any) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("expected JSON object")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return errors.New("invalid message")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("trailing data")
	}
	return nil
}
