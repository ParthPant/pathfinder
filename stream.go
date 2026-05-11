package main

import (
	"io"
	"log/slog"
)

type IStream interface {
	Out() io.Reader
	Err() error
	setErr(err error)
	closeWithError(err error)
}

type PipeStream struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
	Error  error
}

func (ps *PipeStream) Out() io.Reader {
	return ps.Reader
}

func (ps *PipeStream) Err() error {
	return ps.Error
}

func (ps *PipeStream) setErr(err error) {
	ps.Error = err
}

func (ps *PipeStream) closeWithError(err error) {
	closeErr := ps.Writer.CloseWithError(err)
	ps.setErr(err)
	if closeErr != nil {
		slog.Error("Failed to close PipeStream.")
	}
}

func NewPipeStream() PipeStream {
	r, w := io.Pipe()
	return PipeStream{
		Reader: r,
		Writer: w,
		Error:  nil,
	}
}
