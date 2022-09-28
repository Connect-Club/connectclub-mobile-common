package common

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	jvbuster "github.com/Connect-Club/connectclub-jvbuster-client"
	"github.com/Connect-Club/connectclub-mobile-common/datatrack"
	"github.com/Connect-Club/connectclub-mobile-common/volatile"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"
)

type MediaDelegate interface {
	OnPrepare(url, userId string)
	OnPlay(url, userId string)
}

type RecordedRoomState int32

const (
	RecordedRoomStateStarted RecordedRoomState = iota
	RecordedRoomStatePaused
	RecordedRoomStateStopped
)

func (s RecordedRoomState) String() string {
	return [...]string{
		"Started",
		"Paused",
		"Stopped",
	}[s]
}

type RecordedRoom struct {
	BaseRoom
	datatrackReplay *datatrack.Replay
	mediaDelegate   MediaDelegate
	info            infoType
	stopped         chan struct{}
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, errors.New("code!=200")
	}
	return body, nil
}

type infoType struct {
	Datatrack struct {
		Path      string `json:"path"`
		Timestamp int64  `json:"timestamp"`
	} `json:"datatrack"`
	MediaFiles []struct {
		Path      string `json:"path"`
		UserId    string `json:"userId"`
		Timestamp int64  `json:"timestamp"`
	} `json:"mediaFiles"`
}

//todo: add timeout
func ReplayRecordedRoom(
	roomHttpUrl string,
	datatrackDelegate DatatrackDelegate, roomWidthMul, roomHeightMul, adaptiveBubbleSize, devicePixelRatio float64,
	mediaDelegate MediaDelegate,
) (*RecordedRoom, error) {
	infoBody, err := httpGet(fmt.Sprintf("%v/info.json", roomHttpUrl))
	if err != nil {
		return nil, fmt.Errorf("cannot get info.json, reason = %w", err)
	}
	var info infoType
	if err := json.Unmarshal(infoBody, &info); err != nil {
		return nil, fmt.Errorf("cannot unmarshal info, reason = %w", err)
	}

	datatrackData, err := httpGet(fmt.Sprintf("%v/%v", roomHttpUrl, info.Datatrack.Path))
	if err != nil {
		return nil, fmt.Errorf("cannot get datatrack replay data, reason = %w", err)
	}

	replayTimestamp := time.Unix(info.Datatrack.Timestamp, 0)
	timeShift := time.Now().Sub(replayTimestamp)

	datatrackReplay, err := datatrack.NewReplay(
		bytes.NewReader(datatrackData),
		timeShift,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create replay, reason = %w", err)
	}

	defer datatrackDelegate.OnStateChanged(int(RecordedRoomStateStarted))

	r := &RecordedRoom{
		BaseRoom: BaseRoom{
			userId:             volatile.NewValue(""),
			userType:           volatile.NewValue(datatrack.ClientTypeNil),
			datatrackDelegate:  datatrackDelegate,
			roomWidthMul:       roomWidthMul,
			roomHeightMul:      roomHeightMul,
			lastSpeakerId:      "-1",
			lastReactionsState: map[string]string{},
			users:              map[string]Point{},
			onlineAdmins:       jvbuster.StringSet{},
			admins:             jvbuster.StringSet{},
			hands:              jvbuster.StringSet{},
			userSize:           adaptiveBubbleSize * devicePixelRatio,
		},
		datatrackReplay: datatrackReplay,
		mediaDelegate:   mediaDelegate,
		stopped:         make(chan struct{}),
		info:            info,
	}
	go r.datatrackReadLoop()
	go r.mediaEngine(roomHttpUrl, timeShift)
	return r, nil
}

func (r *RecordedRoom) datatrackReadLoop() {
	defer r.datatrackDelegate.OnStateChanged(int(RecordedRoomStateStopped))

	for {
		select {
		case <-r.stopped:
			return
		default:
		}
		typedMessage, rawMessage, preparsedMessage, err := r.datatrackReplay.Read()
		if err != nil {
			if err == datatrack.UnknownMsgTypeErr && rawMessage != nil {
				r.datatrackDelegate.OnMessage(string(rawMessage))
				continue
			} else {
				log.WithError(err).Info("stop datatrack read loop")
				break
			}
		}
		switch typedMessage.(type) {
		case *datatrack.PathResponseMessage:
			r.processPath(typedMessage.(*datatrack.PathResponseMessage))
		case *datatrack.SpeakerResponseMessage:
			r.processSpeaker(typedMessage.(*datatrack.SpeakerResponseMessage))
		case *datatrack.RadarVolumeResponseMessage:
			r.processRadarVolume(typedMessage.(*datatrack.RadarVolumeResponseMessage))
		case *datatrack.BroadcastedResponseMessage:
			r.processBroadcastedMessage(typedMessage.(*datatrack.BroadcastedResponseMessage), preparsedMessage)
		case *datatrack.ConnectionStateResponseMessage:
			r.processConnectionState(typedMessage.(*datatrack.ConnectionStateResponseMessage))
		case *datatrack.StateResponseMessage:
			r.processState(typedMessage.(*datatrack.StateResponseMessage))
		default:
			log.Panicf("not implemented type %v", reflect.TypeOf(typedMessage))
		}
	}
}

func (r *RecordedRoom) playInTime(url, userId string, when time.Time) {
	select {
	case <-r.stopped:
		return
	case <-time.After(when.Sub(time.Now())):
		r.mediaDelegate.OnPlay(url, userId)
	}
}

func (r *RecordedRoom) mediaEngine(roomHttpUrl string, timeShift time.Duration) {
	for _, f := range r.info.MediaFiles {
		url := fmt.Sprintf("%v/%v", roomHttpUrl, f.Path)
		r.mediaDelegate.OnPrepare(url, f.UserId)
		go r.playInTime(url, f.UserId, time.Unix(f.Timestamp, 0).Add(timeShift))
	}
}

func (r *RecordedRoom) Stop() {
	close(r.stopped)
}
