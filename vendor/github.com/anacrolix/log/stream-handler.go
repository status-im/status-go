package log

import (
	"io"
)

type StreamHandler struct {
	W   io.Writer
	Fmt ByteFormatter
}

func (me StreamHandler) Handle(r Record) {
	r.Msg = r.Skip(1)
	me.W.Write(me.Fmt(r))
}

type ByteFormatter func(Record) []byte

func LineFormatter(msg Record) []byte {
	b := []byte{'['}
	beforeLen := len(b)
	b = GetDefaultTimeAppendFormatter()(b)
	if len(b) != beforeLen {
		b = append(b, ' ')
	}
	b = append(b, msg.Level.LogString()...)
	b = append(b, "] "...)
	b = append(b, msg.Text()...)
	b = append(b, " ["...)
	b = append(b, msg.Names[0]...)
	for _, name := range msg.Names[1:] {
		b = append(b, ' ')
		b = append(b, name...)
	}
	b = append(b, ']')
	if b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	return b
}
