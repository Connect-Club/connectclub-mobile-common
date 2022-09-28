package common

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	jvbuster "github.com/Connect-Club/connectclub-jvbuster-client"
	"github.com/Connect-Club/connectclub-mobile-common/datatrack"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
	"github.com/Connect-Club/connectclub-mobile-common/volatile"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strings"
	"sync"
	"time"
)

type JvbusterCallbacks interface {
	UpdateAudioLevel(level int)
	UpdateVideoAudioPhoneState(videoEnabled bool, audioEnabled bool, isOnPhoneCall bool)
}

type JvbusterPeerConnectionCallbacks interface {
	ProcessDataChannelMessage(message string)
	ProcessState(state int)
}

type JvBusterDelegate interface {
	Init(callbacks JvbusterCallbacks)
	InitializeRTCPeerConnection(userId string, id string, isMain bool, isSpeaker bool, sdpOffer string, callbacks JvbusterPeerConnectionCallbacks) string
	SetLocalRTCPeerConnectionDescription(id string, description string)
	SendMessageToDataChannel(id string, message string)
	ProcessMetaAdd(json string)
	ProcessMetaRemove(json string)
	OnError(error string)
	DestroyPeerConnections()
}

type Point struct {
	X, Y float64
}

type Rectangle struct {
	P1, P2 Point
}

func (r Rectangle) IsZero() bool {
	return r.P1.X == r.P2.X && r.P1.Y == r.P2.Y
}

func (r Rectangle) IsInside(point Point) bool {
	return r.P1.X <= point.X && point.X <= r.P2.X &&
		r.P1.Y <= point.Y && point.Y <= r.P2.Y
}

func (r Rectangle) IsInsideWithBorder(point Point, border float64) bool {
	return r.P1.X-border <= point.X && point.X <= r.P2.X+border &&
		r.P1.Y-border <= point.Y && point.Y <= r.P2.Y+border
}

func (r Rectangle) Center() Point {
	return Point{
		X: r.P1.X + (r.P2.X-r.P1.X)/2,
		Y: r.P1.Y + (r.P2.Y-r.P1.Y)/2,
	}
}

type PeerConnectionState int32

const (
	PeerConnectionStateNew PeerConnectionState = iota
	PeerConnectionStateConnecting
	PeerConnectionStateConnected
	PeerConnectionStateDisconnected
	PeerConnectionStateFailed
	PeerConnectionStateClosed
)

func (s PeerConnectionState) String() string {
	return [...]string{
		"New",
		"Connecting",
		"Connected",
		"Disconnected",
		"Failed",
		"Closed",
	}[s]
}

type LiveRoomState int32

const (
	LiveRoomStateConnecting LiveRoomState = iota
	LiveRoomStateConnected
	LiveRoomStateClosed
)

func (s LiveRoomState) String() string {
	return [...]string{
		"Connecting",
		"Connected",
		"Closed",
	}[s]
}

type LiveRoom struct {
	BaseRoom
	state                 LiveRoomState
	datatrackConn         *datatrack.Connection
	datatrackState        datatrack.ConnectionState
	datatrackDone         chan struct{}
	participants          *volatile.Value[[]string]
	screenPosition        Rectangle
	isFullScreen          *volatile.Value[bool]
	mode                  *volatile.Value[string]
	audioLevel            int
	videoEnabled          bool
	audioEnabled          bool
	isOnPhoneCall         bool
	jvbusterClient        *jvbuster.Client
	jvbusterDelegate      JvBusterDelegate
	jvbusterSubscribeTask *jvbuster.Task
	jvbusterStartTask     *jvbuster.Task
	//todo: store state for each peer connection
	jvbusterPeerConnectionState PeerConnectionState
	fullResCircleDiameterInch   float64
	viewportWidthInch           float64
	viewport                    Rectangle
	visibleParticipants         jvbuster.StringSet
	adaptiveBubbleSize          float64
	allSdpOffers                map[ /*PeerConnectionId*/ string]map[ /*StreamId*/ string]jvbuster.SdpOfferMeta

	mu sync.RWMutex
}

func ConnectToLiveRoom(
	datatrackDelegate DatatrackDelegate, wsUrl, roomId, accessToken, roomPass string, roomWidthMul, roomHeightMul, adaptiveBubbleSize, screenPositionX, screenPositionY, screenPositionWidth, screenPositionHeight, devicePixelRatio float64,
	jvbusterDelegate JvBusterDelegate, jvbusterAddress, jvbusterToken string, fullResCircleDiameterInch, viewportWidthInch float64, videoBandwidth, audioBandwidth int,
) (*LiveRoom, error) {
	var err error

	r := &LiveRoom{
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
		state:                     LiveRoomStateConnecting,
		datatrackConn:             nil, //init later,
		datatrackState:            datatrack.ConnectionStateUndefined,
		datatrackDone:             make(chan struct{}),
		participants:              volatile.NewValue([]string{}),
		screenPosition:            Rectangle{P1: Point{X: screenPositionX, Y: screenPositionY}, P2: Point{X: screenPositionX + screenPositionWidth, Y: screenPositionY + screenPositionHeight}},
		isFullScreen:              volatile.NewValue(false),
		mode:                      volatile.NewValue(""),
		jvbusterClient:            nil, //init later,
		jvbusterDelegate:          jvbusterDelegate,
		jvbusterSubscribeTask:     nil, //init later
		jvbusterStartTask:         nil, //init later
		fullResCircleDiameterInch: fullResCircleDiameterInch,
		viewportWidthInch:         viewportWidthInch,
		adaptiveBubbleSize:        adaptiveBubbleSize,
		allSdpOffers:              make(map[ /*PeerConnectionId*/ string]map[ /*StreamId*/ string]jvbuster.SdpOfferMeta),
	}

	r.jvbusterStartTask = jvbuster.CreateTask(r.jvbusterStart, 0, false)
	r.jvbusterSubscribeTask = jvbuster.CreateTask(r.jvbusterSubscribe, 0, true)

	r.datatrackConn = datatrack.NewConnection(wsUrl, roomId, roomPass, accessToken, r.onDatatrackUserId, r.onDatatrackState)

	r.jvbusterClient, err = jvbuster.NewClient(
		jvbusterAddress,
		jvbusterToken,
		r.onJvbusterNewMessageForDataChannel,
		r.onJvbusterNewSdpOffer,
		r.onJvbusterExpired,
		nil, // used in web
		nil, // used in web
		videoBandwidth,
		audioBandwidth,
	)
	if err != nil {
		//todo: throw custom error
		return nil, err
	}

	jvbusterDelegate.Init(r)

	go r.datatrackReadLoop()
	go r.stateSenderLoop()

	go datatrackDelegate.OnStateChanged(int(LiveRoomStateConnecting))
	return r, nil
}

func (r *LiveRoom) Disconnect() {
	log.Info("⤵")
	defer log.Info("⤴")

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == LiveRoomStateClosed {
		log.Info("the room is already closed")
		return
	}

	r.state = LiveRoomStateClosed

	r.datatrackConn.Close()

	if err := r.jvbusterStartTask.Stop(time.Second * 10); err != nil {
		log.WithError(err).Error("cannot stop jvbusterStartTask")
	}
	r.jvbusterClient.Stop()
	go r.jvbusterClient.Destroy()

	r.jvbusterDelegate.DestroyPeerConnections()

	log.Infof("room state=%v", r.state.String())
	r.datatrackDelegate.OnStateChanged(int(r.state))
}

func (r *LiveRoom) SetViewport(x1, y1, x2, y2 float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	r.viewport = Rectangle{P1: Point{X: x1, Y: y1}, P2: Point{X: x2, Y: y2}}
	r.jvbusterSubscribeTask.Run()
}

func (r *LiveRoom) SetJvbusterSubscriptionType(subscriptionType string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	switch subscriptionType {
	case "normalSubscription":
		r.jvbusterClient.SetSubscriptionType(jvbuster.NormalSubscription)
	case "audioSubscription":
		r.jvbusterClient.SetSubscriptionType(jvbuster.AudioSubscription)
	case "mixedAudioSubscription":
		r.jvbusterClient.SetSubscriptionType(jvbuster.MixedAudioSubscription)
	default:
		log.Panicf("unsupported subscription type: %v", subscriptionType)
	}
}

func (r *LiveRoom) createProcessDataChannelMessageCallback(peerConnectionId string) func(message string) {
	return func(message string) {
		r.ProcessDataChannelMessage(peerConnectionId, message)
	}
}

func (r *LiveRoom) ProcessDataChannelMessage(peerConnectionId, message string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	if err := r.jvbusterClient.ProcessDataChannelMessage(peerConnectionId, message); err != nil {
		if err == jvbuster.InactiveClientError {
			log.Info("ProcessDataChannelMessage called on inactive client")
		} else {
			log.WithError(err).Error("cannot process datachannel message")
		}
	}
}

func (r *LiveRoom) createProcessPeerConnectionStateCallback(peerConnectionId string) func(state int) {
	return func(state int) {
		r.ProcessPeerConnectionState(peerConnectionId, state)
	}
}

func (r *LiveRoom) ProcessPeerConnectionState(peerConnectionId string, state int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	if _, ok := r.allSdpOffers[peerConnectionId]; !ok {
		return
	}

	r.jvbusterPeerConnectionState = PeerConnectionState(state)
	log.Infof("peerconnection id=%v state=%v", peerConnectionId, PeerConnectionState(state).String())
	if r.jvbusterPeerConnectionState == PeerConnectionStateFailed {
		r.jvbusterStartTask.Run()
	}
	r.processStates()
}

func (r *LiveRoom) UpdateAudioLevel(level int) {
	r.audioLevel = level
}

func (r *LiveRoom) UpdateVideoAudioPhoneState(videoEnabled bool, audioEnabled bool, isOnPhoneCall bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	r.videoEnabled = videoEnabled
	r.audioEnabled = audioEnabled
	r.isOnPhoneCall = isOnPhoneCall

	r.sendState()
}

func (r *LiveRoom) SendUserPath(toX float64, toY float64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	pointX := toX * (1 / r.roomWidthMul)
	pointY := toY * (1 / r.roomHeightMul)
	r.datatrackConn.SendPath(0, 0, pointX, pointY, "move")
}

func (r *LiveRoom) RemoveReaction(id string) {
	delete(r.lastReactionsState, id)
}

func (r *LiveRoom) SendMessage(message string) {
	r.datatrackConn.SendMessage(message)
}

func (r *LiveRoom) FullScreen(val bool) {
	r.isFullScreen.Store(val)

	r.jvbusterSubscribeTask.Run()
}

func (r *LiveRoom) sendState() {
	if r.mode.Load() != "room" {
		return
	}

	r.datatrackConn.SendState(r.videoEnabled, r.audioEnabled, r.isOnPhoneCall)
	if r.audioEnabled {
		r.datatrackConn.SendAudioLevel(r.audioLevel)
	} else {
		r.datatrackConn.SendAudioLevel(0)
	}
}

func (r *LiveRoom) jvbusterStart(ctx context.Context) {
	log.Info("⤵")
	defer log.Info("⤴")

	r.jvbusterClient.Stop()
	r.jvbusterDelegate.DestroyPeerConnections()
	for k := range r.allSdpOffers {
		delete(r.allSdpOffers, k)
	}

	r.mu.Lock()
	r.jvbusterPeerConnectionState = PeerConnectionStateClosed
	r.processStates()
	r.mu.Unlock()

	var speaker bool
	var guestEndpoint string
	if r.userType.Load() == datatrack.ClientTypeNormal {
		// only normal users have ability to speak
		speaker = r.mode.Load() == "room"
	} else {
		speaker = false
		// abnormal users usually have id generated in datatrack
		guestEndpoint = r.userId.Load()
	}

	if err := r.jvbusterClient.Start(ctx, speaker, guestEndpoint); err != nil {
		log.WithError(err).Error("can not start jvbuster client")
		if !errors.Is(err, ctx.Err()) {
			r.handleJvbusterError(err)
		}
	}
}

func (r *LiveRoom) setMode(mode string) {
	if len(mode) == 0 {
		log.Warn("trying to set empty mode")
		return
	}
	previousMode := r.mode.Swap(mode)

	if !strings.EqualFold(previousMode, mode) {
		log.Infof("new mode=%v", mode)
		r.datatrackDelegate.OnChangeRoomMode(mode, len(previousMode) == 0)
	}

	if !r.jvbusterClient.IsActive() || r.jvbusterClient.IsSpeaker() != (mode == "room") {
		r.jvbusterStartTask.Run()
	}
}

func (r *LiveRoom) datatrackReadLoop() {
	for {
		typedMessage, rawMessage, preparsedMessage, err := r.datatrackConn.Read()
		if err != nil {
			if err == datatrack.UnknownMsgTypeErr && rawMessage != nil {
				r.datatrackDelegate.OnMessage(string(rawMessage))
				continue
			} else if err == datatrack.Closed {
				log.Info("stop datatrack read loop")
				break
			} else {
				log.WithError(err).Warn("cannot read message from datatrack")
				time.Sleep(time.Second)
				continue
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

func (r *LiveRoom) stateSenderLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		r.mu.RLock()
		if r.state == LiveRoomStateClosed {
			r.mu.RUnlock()
			return
		}
		r.sendState()
		r.mu.RUnlock()
	}
}

func (r *LiveRoom) processRadarVolume(msg *datatrack.RadarVolumeResponseMessage) {
	r.participants.Store(msg.Participants)
	r.jvbusterSubscribeTask.Run()

	r.BaseRoom.processRadarVolume(msg)
}

func (r *LiveRoom) processConnectionState(msg *datatrack.ConnectionStateResponseMessage) {
	if msg.ID == r.datatrackConn.UserId {
		r.setMode(msg.Mode)
	}

	r.BaseRoom.processConnectionState(msg)
}

func (r *LiveRoom) processState(msg *datatrack.StateResponseMessage) {
	if len(msg.Current.ID) == 0 {
		return
	}

	r.setMode(msg.Current.Mode)

	r.BaseRoom.processState(msg)
}

func (r *LiveRoom) onDatatrackUserId(userId string, userType datatrack.ClientType) {
	log.Infof("userId=%v userType=%v received from datatrack", userId, userType)
	r.BaseRoom.userId.Store(userId)
	r.BaseRoom.userType.Store(userType)
}

func (r *LiveRoom) onDatatrackState(newState datatrack.ConnectionState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state == LiveRoomStateClosed {
		return
	}

	r.datatrackState = newState
	log.Infof("datatrack state=%v", newState.String())
	r.processStates()
}

func (r *LiveRoom) processStates() {
	var newState LiveRoomState
	if r.datatrackState == datatrack.ConnectionStateConnected && r.jvbusterPeerConnectionState == PeerConnectionStateConnected && r.jvbusterClient.IsActive() {
		newState = LiveRoomStateConnected
	} else if r.datatrackState == datatrack.ConnectionStateClosed && r.jvbusterPeerConnectionState == PeerConnectionStateClosed && !r.jvbusterClient.IsActive() {
		newState = LiveRoomStateClosed
	} else {
		newState = LiveRoomStateConnecting
	}
	if r.state != newState {
		r.state = newState
		log.Infof("room state=%v", newState.String())
		r.datatrackDelegate.OnStateChanged(int(newState))
	}
}

func (r *LiveRoom) onJvbusterNewMessageForDataChannel(peerConnectionId string, msg string) {
	r.jvbusterDelegate.SendMessageToDataChannel(peerConnectionId, msg)
}

func (r *LiveRoom) onJvbusterNewSdpOffer(sdpOffers map[ /*PeerConnectionId*/ string]jvbuster.SdpOffer, accepted func() error) {
	if len(sdpOffers) == 0 {
		err := accepted()
		if err != nil {
			r.handleJvbusterError(err)
		}
		return
	}

	for pcId, value := range sdpOffers {
		r.allSdpOffers[pcId] = value.Meta
	}

	metaForProcess := utils.H{}
	for _, meta := range r.allSdpOffers {
		for remoteUserId, mediaStream := range meta {
			var video string
			var audio string
			if len(mediaStream.VideoTracks) > 0 {
				video = mediaStream.VideoTracks[0]
			}

			if len(mediaStream.AudioTracks) > 0 {
				audio = mediaStream.AudioTracks[0]
			}
			metaForProcess[remoteUserId] = utils.H{
				"video": video,
				"audio": audio,
			}
		}
	}
	metaJson, _ := json.Marshal(metaForProcess)
	r.jvbusterDelegate.ProcessMetaRemove(string(metaJson))

	// create or update peerConnections
	var createdSdps = map[string]string{}
	for key, value := range sdpOffers {
		// skip all empty sdp
		if len(value.Text) == 0 {
			continue
		}
		createdSdp := r.jvbusterDelegate.InitializeRTCPeerConnection(
			r.userId.Load(),
			key,
			value.Primary,
			r.jvbusterClient.IsSpeaker(),
			value.Text,
			r.createPeerConnectionCallbacks(key),
		)
		if strings.EqualFold(createdSdp, "") {
			log.Info("exit from app after receive empty sdp")
			return
		}
		log.Infof("InitializeRTCPeerConnection peerConnectionId=%s sdp=%s", key, base64.StdEncoding.EncodeToString([]byte(createdSdp)))
		createdSdps[key] = createdSdp
	}

	if len(createdSdps) > 0 {
		sdpProcessed, err := r.jvbusterClient.ModifyAndSendAnswers(createdSdps)
		if err != nil {
			if err == jvbuster.InactiveClientError {
				log.Info("jvBusterClient.ModifyAndSendAnswers called on inactive client")
			} else if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("jvBusterClient.ModifyAndSendAnswers deadline")
				r.jvbusterStartTask.Run()
			} else {
				log.WithError(err).Error("jvBusterClient.ModifyAndSendAnswers")
				r.handleJvbusterError(err)
			}
			return
		}
		log.Info("jvBusterClient.SetLocalRTCPeerConnectionDescription")
		for key, value := range sdpProcessed {
			log.Infof("SetLocalRTCPeerConnectionDescription peerConnectionId=%s sdp=%s", key, base64.StdEncoding.EncodeToString([]byte(value)))
			r.jvbusterDelegate.SetLocalRTCPeerConnectionDescription(key, value)
		}
		log.Info("jvBusterClient.SetLocalRTCPeerConnectionDescription complete")
	}

	r.jvbusterDelegate.ProcessMetaAdd(string(metaJson))

	if err := accepted(); err != nil {
		r.handleJvbusterError(err)
	}
	// createdSdps empty means process destroying
	if len(createdSdps) != 0 {
		log.Info("afterAccepted")
	}
}

func (r *LiveRoom) onJvbusterExpired() {
	r.jvbusterDelegate.OnError("expired")
}

func (r *LiveRoom) jvbusterSubscribe(_ context.Context) {
	//get copies because they might be modified
	participants, viewport := r.participants.Load(), r.viewport
	screenPosition, isFullScreen := r.screenPosition, r.isFullScreen.Load()

	subscribedEndpoints := make(map[string]jvbuster.VideoConstraint)
	shownParticipants := make([]string, 0, len(participants))
	hiddenParticipants := make([]string, 0, len(participants))
	newVisibleParticipants := jvbuster.NewStringSetSized(len(participants))
	for _, participant := range participants {
		insideViewport := true
		lowRes := false
		if !viewport.IsZero() {
			if isFullScreen {
				insideViewport = false
				lowRes = true
			} else {
				position, ok := r.users[participant]
				if ok {
					if viewport.IsInside(position) {
						lowRes = (r.viewportWidthInch * r.adaptiveBubbleSize / (viewport.P2.X - viewport.P1.X)) <= r.fullResCircleDiameterInch
					} else {
						insideViewport = false
						lowRes = true
					}
				}
			}
		}
		subscribedEndpoints[participant] = jvbuster.VideoConstraint{
			OffVideo: false,
			LowRes:   lowRes,
			LowFps:   !insideViewport,
		}
		newVisibleParticipants.Add(participant)
		if !r.visibleParticipants.Contains(participant) {
			shownParticipants = append(shownParticipants, participant)
		}
	}
	for oldVisibleParticipant, _ := range r.visibleParticipants {
		if !newVisibleParticipants.Contains(oldVisibleParticipant) {
			hiddenParticipants = append(hiddenParticipants, oldVisibleParticipant)
		}
	}

	screenVideoConstraint := jvbuster.VideoConstraint{}

	if !viewport.IsZero() && !screenPosition.IsZero() && !isFullScreen {
		screenVideoConstraint.LowRes = !viewport.IsInside(screenPosition.Center()) || !isScreenCloseToViewport(r.viewport, screenPosition, 0.2)
		screenVideoConstraint.LowFps = !viewport.IsInside(screenPosition.Center())
	}

	r.jvbusterClient.Subscribe(subscribedEndpoints, screenVideoConstraint)
	r.visibleParticipants = newVisibleParticipants

	if len(shownParticipants) > 0 || len(hiddenParticipants) > 0 {
		jsonString := utils.PackToJsonString(utils.H{
			"shown":  shownParticipants,
			"hidden": hiddenParticipants,
		})
		r.datatrackDelegate.OnParticipantsVisibilityChanged(jsonString)
	}
}

func (r *LiveRoom) handleJvbusterError(err error) {
	if errors.Is(err, jvbuster.UnknownPeerConnectionError) || errors.Is(err, jvbuster.AllOffersHaveToBeAcceptedError) {
		log.WithError(err).Panic()
	} else {
		r.jvbusterDelegate.OnError(err.Error())
	}
}

type peerConnectionCallbacks struct {
	liveRoom         *LiveRoom
	peerConnectionId string
}

func (c *peerConnectionCallbacks) ProcessDataChannelMessage(message string) {
	c.liveRoom.ProcessDataChannelMessage(c.peerConnectionId, message)
}

func (c *peerConnectionCallbacks) ProcessState(state int) {
	c.liveRoom.ProcessPeerConnectionState(c.peerConnectionId, state)
}

func (r *LiveRoom) createPeerConnectionCallbacks(peerConnectionId string) *peerConnectionCallbacks {
	return &peerConnectionCallbacks{
		liveRoom:         r,
		peerConnectionId: peerConnectionId,
	}
}

func isScreenCloseToViewport(viewport, screen Rectangle, freeMarginThreshold float64) bool {
	leftMargin := screen.P1.X - viewport.P1.X
	if leftMargin < 0 {
		leftMargin = 0
	}
	topMargin := screen.P1.Y - viewport.P1.Y
	if topMargin < 0 {
		topMargin = 0
	}
	rightMargin := viewport.P2.X - screen.P2.X
	if rightMargin < 0 {
		rightMargin = 0
	}
	bottomMargin := viewport.P2.Y - screen.P2.Y
	if bottomMargin < 0 {
		bottomMargin = 0
	}
	horizontalMarginFraction := (leftMargin + rightMargin) / (viewport.P2.X - viewport.P1.X)
	verticalMarginFraction := (topMargin + bottomMargin) / (viewport.P2.Y - viewport.P1.Y)
	return horizontalMarginFraction <= freeMarginThreshold || verticalMarginFraction <= freeMarginThreshold
}
