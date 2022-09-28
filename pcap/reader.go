package pcap

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

func createPacketReaderChannel(ctxLog *log.Entry, reader io.Reader, headerSize int, newPacketFn func() packet) (chan packet, error) {
	ch := make(chan packet)
	go func() {
		defer close(ch)

		headerBuf := make([]byte, headerSize)

		for {
			pkt := newPacketFn()
			if err := readPacket(reader, headerBuf, pkt); err != nil {
				if err != io.EOF {
					ctxLog.WithError(err).Warn("can not read packet")
				}
				return
			}
			ch <- pkt
		}
	}()
	return ch, nil
}

type SimpleReader struct {
	ctxLog   *log.Entry
	packetCh chan packet
}

func NewSimpleReader(ctxLog *log.Entry, reader io.Reader) (*SimpleReader, error) {
	ctxLog = ctxLog.WithField("readerType", "simple")
	packetCh, err := createPacketReaderChannel(ctxLog, reader, simpleHeaderSize, newSimplePacket)
	if err != nil {
		return nil, fmt.Errorf("can not create packet reader channel, err: %w", err)
	}

	return &SimpleReader{
		ctxLog:   ctxLog,
		packetCh: packetCh,
	}, nil
}

func (r *SimpleReader) Read() (p []byte, capturedTime time.Time, err error) {
	pkt, ok := <-r.packetCh
	if !ok {
		return nil, time.Time{}, io.EOF
	}

	simplePkt, ok := pkt.(*simplePacket)
	if !ok {
		r.ctxLog.Panic("not a simple packet")
	}

	return simplePkt.payload, simplePkt.capturedTime, nil
}
