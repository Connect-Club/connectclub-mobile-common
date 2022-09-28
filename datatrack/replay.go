package datatrack

import (
	"encoding/json"
	"fmt"
	"github.com/Connect-Club/connectclub-mobile-common/pcap"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

type Replay struct {
	timeShift          time.Duration
	simplePacketReater *pcap.SimpleReader
}

func NewReplay(reader io.Reader, timeShift time.Duration) (*Replay, error) {
	simplePacketReater, err := pcap.NewSimpleReader(logrus.NewEntry(logrus.New()), reader)
	if err != nil {
		return nil, fmt.Errorf("cannot create simple packet reater, reason = %w", err)
	}

	r := &Replay{
		timeShift:          timeShift,
		simplePacketReater: simplePacketReater,
	}
	return r, nil
}

func (r *Replay) Read() (typedMessage interface{}, rawMessage []byte, preparsedMessage map[string]json.RawMessage, err error) {
	rawMessage, capturedTime, err := r.simplePacketReater.Read()
	if err != nil {
		return
	} else {
		time.Sleep(capturedTime.Add(r.timeShift).Sub(time.Now()))
	}
	typedMessage, preparsedMessage, err = extractPayload(rawMessage)
	return
}
