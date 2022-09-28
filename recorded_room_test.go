package common

import (
	"fmt"
	"testing"
	"time"
)

type TestRecordedRoomDatatrackDelegate struct {
	done chan struct{}
}

func (t *TestRecordedRoomDatatrackDelegate) OnStateChanged(newState int) {
	state := RecordedRoomState(newState)
	fmt.Println("DatatrackDelegate.OnStateChanged", state)
	if state == RecordedRoomStateStopped {
		close(t.done)
	}
}

func (t *TestRecordedRoomDatatrackDelegate) OnMessage(json string) {
	fmt.Println("DatatrackDelegate.OnMessage", json)
}

func (t *TestRecordedRoomDatatrackDelegate) OnReaction(json []byte) {
	fmt.Println("DatatrackDelegate.OnReaction", string(json))
}

func (t *TestRecordedRoomDatatrackDelegate) OnPath(userId string, x float64, y float64, duration float64) {
	fmt.Println("DatatrackDelegate.OnPath", userId, x, y, duration)
}

func (t *TestRecordedRoomDatatrackDelegate) OnNativeState(json []byte) {
	fmt.Println("DatatrackDelegate.OnNativeState", string(json))
}

func (t *TestRecordedRoomDatatrackDelegate) OnRadarVolume(json []byte) {
	fmt.Println("DatatrackDelegate.OnRadarVolume", string(json))
}

func (t *TestRecordedRoomDatatrackDelegate) OnConnectionState(json []byte) {
	fmt.Println("DatatrackDelegate.OnConnectionState", string(json))
}

func (t *TestRecordedRoomDatatrackDelegate) OnPopupUsers(json string) {
	fmt.Println("DatatrackDelegate.OnPopupUsers", json)
}

func (t *TestRecordedRoomDatatrackDelegate) OnChangeRoomMode(mode string, isFirstConnection bool) {
	fmt.Println("DatatrackDelegate.OnChangeRoomMode", mode, isFirstConnection)
}

func (t *TestRecordedRoomDatatrackDelegate) OnParticipantsVisibilityChanged(json string) {
	fmt.Println("DatatrackDelegate.OnParticipantsVisibilityChanged", json)
}

type TestMediaDelegate struct{}

func (t *TestMediaDelegate) OnPrepare(url, userId string) {
	fmt.Println("MediaDelegate.OnPrepare", url, userId)
}

func (t *TestMediaDelegate) OnPlay(url, userId string) {
	fmt.Println("MediaDelegate.onPlay", url, userId)
}

func TestReplayRecordedRoom(t *testing.T) {
	datatrackDelegate := &TestRecordedRoomDatatrackDelegate{
		done: make(chan struct{}),
	}
	mediaDelegate := &TestMediaDelegate{}

	recordedRoom, err := ReplayRecordedRoom(
		"https://storage.googleapis.com/recorder-files-public-videobridge-stage/6257ffb7d9da3",
		datatrackDelegate, 1.0, 1.0, 1.0, 1.0,
		mediaDelegate,
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("start")

	select {
	case <-datatrackDelegate.done:
	case <-time.After(time.Minute * 5):
		recordedRoom.Stop()
		select {
		case <-datatrackDelegate.done:
		case <-time.After(time.Second):
			panic("recorded room does not want to be stopped")
		}
	}

	fmt.Println("stop")
}
