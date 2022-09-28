package common

import (
	"encoding/json"
	jvbuster "github.com/Connect-Club/connectclub-jvbuster-client"
	"github.com/Connect-Club/connectclub-mobile-common/datatrack"
	"github.com/Connect-Club/connectclub-mobile-common/utils"
	"github.com/Connect-Club/connectclub-mobile-common/volatile"
	"strings"
)

type DatatrackDelegate interface {
	OnMessage(json string)
	OnReaction(json []byte)
	OnPath(userId string, x float64, y float64, duration float64)
	OnNativeState(json []byte)
	OnRadarVolume(json []byte)
	OnConnectionState(json []byte)
	OnPopupUsers(json string)
	OnChangeRoomMode(mode string, isFirstConnection bool)
	OnParticipantsVisibilityChanged(json string)
	OnStateChanged(newState int)
}

type BaseRoom struct {
	userId                      *volatile.Value[string]
	userType                    *volatile.Value[datatrack.ClientType]
	datatrackDelegate           DatatrackDelegate
	roomWidthMul, roomHeightMul float64
	lastSpeakerId               string
	lastReactionsState          map[string]string
	users                       map[string]Point
	onlineAdmins                jvbuster.StringSet
	admins                      jvbuster.StringSet
	hands                       jvbuster.StringSet
	userSize                    float64
}

func (r *BaseRoom) Hands() []byte {
	return utils.PackToByteArray(r.hands.GetSlice())
}

func (r *BaseRoom) Admins() []byte {
	return utils.PackToByteArray(r.admins.GetSlice())
}

func (r *BaseRoom) IsThereOtherAdmin() bool {
	onlineAdmins := r.onlineAdmins
	count := len(onlineAdmins)
	if count == 0 {
		return false
	}
	return count > 1 || !onlineAdmins.Contains(r.userId.Load())
}

func (r *BaseRoom) convertRawPointToRelative(x, y float64) (float64, float64) {
	return x * r.roomWidthMul, y * r.roomHeightMul
}

func (r *BaseRoom) getReaction(userId string) string {
	reaction := r.lastReactionsState[userId]
	if len(reaction) == 0 {
		reaction = "none"
	}
	return reaction
}

func (r *BaseRoom) processPath(msg *datatrack.PathResponseMessage) {
	var totalDuration float64 = 0
	for _, p := range msg.Points {
		totalDuration += p.Duration
	}
	lastPoint := msg.Points[len(msg.Points)-1]
	x, y := r.convertRawPointToRelative(lastPoint.X, lastPoint.Y)
	r.users[msg.ID] = Point{X: x, Y: y}
	r.datatrackDelegate.OnPath(msg.ID, x, y, totalDuration/1000.0)
}

func (r *BaseRoom) processSpeaker(msg *datatrack.SpeakerResponseMessage) {
	if msg.ID == "0" && r.lastSpeakerId == "-1" {
		return
	}
	if msg.ID == r.lastSpeakerId {
		return
	}
	previousId := r.lastSpeakerId
	if previousId != "-1" {
		r.datatrackDelegate.OnMessage(utils.PackToJsonString(utils.H{
			"type":    "speaker",
			"payload": utils.H{"id": previousId, "isSpeaker": false},
		}))
	}

	r.datatrackDelegate.OnMessage(utils.PackToJsonString(utils.H{
		"type":    "speaker",
		"payload": utils.H{"id": msg.ID, "isSpeaker": true},
	}))

	r.datatrackDelegate.OnMessage(utils.PackToJsonString(utils.H{
		"type":    "showSpeakerNotification",
		"payload": msg.ID,
	}))
	r.lastSpeakerId = msg.ID
}

func (r *BaseRoom) processRadarVolume(msg *datatrack.RadarVolumeResponseMessage) {
	r.datatrackDelegate.OnRadarVolume(utils.PackToByteArray(utils.H{
		"radarVolume":  msg.RadarVolume,
		"screenVolume": msg.ScreenVolume,
		"isSubscriber": msg.IsSubscriber,
	}))
}

//todo: !!!need refactoring!!!
func (r *BaseRoom) processBroadcastedMessage(broadcastedMessage *datatrack.BroadcastedResponseMessage, preparsedMessage map[string]json.RawMessage) {
	msg := *broadcastedMessage

	var userId string
	_ = json.Unmarshal(preparsedMessage["from"], &userId)
	broadcastType := msg["type"].(string)

	if broadcastType == "nonverbal" {
		reactionType := msg["message"].(string)
		r.lastReactionsState[userId] = reactionType
		reactions := utils.H{
			"type": "reactions",
			"payload": utils.H{
				"fromId":   userId,
				"reaction": reactionType,
				"duration": msg["duration"],
			},
		}
		r.datatrackDelegate.OnReaction(utils.PackToByteArray(reactions))
		r.datatrackDelegate.OnMessage(utils.PackToJsonString(reactions))
	} else if broadcastType == "timer" {
		timer := utils.H{
			"type": "timer",
			"payload": utils.H{
				"fromId":        userId,
				"duration":      msg["duration"],
				"startUserName": msg["startUserName"],
			},
		}
		r.datatrackDelegate.OnMessage(utils.PackToJsonString(timer))
	} else if broadcastType == "shareDesktop" {
		shareDesktop := utils.H{
			"type": "shareDesktop",
			"payload": utils.H{
				"fromId":    userId,
				"isSharing": msg["isSharing"],
			},
		}
		r.datatrackDelegate.OnMessage(utils.PackToJsonString(shareDesktop))
	}
}

func (r *BaseRoom) processConnectionState(msg *datatrack.ConnectionStateResponseMessage) {
	r.datatrackDelegate.OnConnectionState(utils.PackToByteArray(utils.H{
		"id":    msg.ID,
		"mode":  msg.Mode,
		"state": msg.State,
	}))
}

func (r *BaseRoom) processState(msg *datatrack.StateResponseMessage) {
	nativeState := make([]utils.H, 0)

	r.onlineAdmins = jvbuster.NewStringSetFromSlice(msg.OnlineAdmins)
	r.admins = jvbuster.NewStringSetFromSlice(msg.Admins)
	r.hands = jvbuster.NewStringSetFromSlice(msg.Hands)

	deviceState := DeviceState{
		Current:                nil,
		Room:                   []*DeviceRoomUser{},
		ListenersCount:         msg.ListenersCount,
		RaisedHandsCount:       len(msg.Hands),
		HandsAllowed:           msg.HandsAllowed,
		AbsoluteSpeakerPresent: msg.AbsoluteSpeakerPresent,
	}

	if msg.Current.ID == r.userId.Load() {
		nativeState = append(nativeState, r.processParticipant(&msg.Current))
		deviceState.Current = &DeviceCurrentUser{
			IsAdmin:           msg.Current.IsAdmin,
			IsHandRaised:      r.hands.Contains(msg.Current.ID),
			Mode:              msg.Current.Mode,
			IsAbsoluteSpeaker: msg.Current.IsAbsoluteSpeaker,
		}
	}
	for _, u := range msg.Room {
		nativeState = append(nativeState, r.processParticipant(&u))
		isLocal := strings.EqualFold(u.ID, r.userId.Load())
		deviceState.Room = append(deviceState.Room, &DeviceRoomUser{
			Id:                u.ID,
			Size:              r.userSize,
			IsLocal:           isLocal,
			HasRadar:          isLocal,
			Name:              u.Name,
			Surname:           u.Surname,
			Avatar:            u.Avatar,
			Mode:              u.Mode,
			IsAdmin:           u.IsAdmin,
			IsExpired:         u.Expired,
			InRadar:           u.InRadar,
			Video:             u.Video,
			Audio:             u.Audio,
			PhoneCall:         u.PhoneCall,
			IsSpecialGuest:    u.IsSpecialGuest,
			IsAbsoluteSpeaker: u.IsAbsoluteSpeaker,
			Badges:            u.Badges,
		})
	}
	r.datatrackDelegate.OnNativeState(utils.PackToByteArray(nativeState))
	r.processPopup(len(msg.Room), msg.ListenersCount, msg.Popup)

	r.datatrackDelegate.OnMessage(utils.PackToJsonString(utils.H{
		"type":    "state",
		"payload": deviceState,
	}))
}

func (r *BaseRoom) processParticipant(p *datatrack.Participant) utils.H {
	audioLevel := 1 + float64(p.AudioLevel)/100
	x, y := r.convertRawPointToRelative(p.X, p.Y)
	r.users[p.ID] = Point{X: x, Y: y}
	return utils.H{
		"isLocal":           strings.EqualFold(p.ID, r.userId.Load()),
		"id":                p.ID,
		"x":                 x,
		"y":                 y,
		"audioLevel":        audioLevel,
		"video":             p.Video,
		"audio":             p.Audio,
		"phoneCall":         p.PhoneCall,
		"avatar":            p.Avatar,
		"isExpired":         p.Expired,
		"isAdmin":           p.IsAdmin,
		"isOwner":           p.IsOwner,
		"isSpeaker":         p.IsSpeaker,
		"isSpecialGuest":    p.IsSpecialGuest,
		"name":              p.Name,
		"surname":           p.Surname,
		"reaction":          r.getReaction(p.ID),
		"inRadar":           p.InRadar,
		"mode":              p.Mode,
		"badges":            p.Badges,
		"isHandRaised":      r.hands.Contains(p.ID),
		"isAbsoluteSpeaker": p.IsAbsoluteSpeaker,
		"size":              r.userSize,
	}
}

func (r *BaseRoom) processPopup(speakersCount, listenersCount int, popupUsers []datatrack.ShortParticipant) {
	for i, user := range popupUsers {
		user.Reaction = r.getReaction(user.ID)
		user.IsLocal = strings.EqualFold(user.ID, r.userId.Load())
		popupUsers[i] = user
	}

	r.datatrackDelegate.OnPopupUsers(utils.PackToJsonString(utils.H{
		"speakersCount":  speakersCount,
		"listenersCount": listenersCount,
		"users":          popupUsers,
	}))
}
