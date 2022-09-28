package datatrack

import (
	"encoding/json"
	"fmt"
)

type ClientType int

const (
	ClientTypeNil ClientType = iota
	ClientTypeNormal
	ClientTypeGuest
	ClientTypeService
)

var ClientTypeStrings = [...]string{"Nil", "Normal", "Guest", "Service"}

func (t ClientType) String() string {
	return ClientTypeStrings[t]
}

func ClientTypeFromString(s string) (ClientType, error) {
	for i, v := range ClientTypeStrings {
		if v == s {
			return ClientType(i), nil
		}
	}
	return -1, fmt.Errorf("unknown ClientType(%v)", s)
}

func (t ClientType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *ClientType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	} else if clientType, err := ClientTypeFromString(s); err != nil {
		return err
	} else {
		*t = clientType
		return nil
	}
}

type RegisterResponseMessage struct {
	ID   string     `json:"id"`
	Type ClientType `json:"type"`
}

type PathResponseMessage struct {
	ID     string      `json:"id"`
	Points []PathPoint `json:"points"`
}

type PathPoint struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Duration float64 `json:"duration"`
	Index    int     `json:"index"`
}

type SpeakerResponseMessage struct {
	ID string `json:"id"`
}

type RadarVolumeResponseMessage struct {
	Participants []string            `json:"participants"`
	IsSubscriber bool                `json:"isSubscriber"`
	RadarVolume  []radarVolumeStruct `json:"radarVolume"`
	ScreenVolume json.Number         `json:"screenVolume"`
}

type radarVolumeStruct struct {
	Volume json.Number `json:"volume"`
	ID     string      `json:"id"`
}

//todo: !!!need refactoring!!!
type BroadcastedResponseMessage map[string]interface{}

type ConnectionStateResponseMessage struct {
	ID    string           `json:"id"`
	State string           `json:"state"`
	Mode  string           `json:"mode"`
	User  ShortParticipant `json:"user"`
}

type ShortParticipant struct {
	Name           string   `json:"name"`
	Surname        string   `json:"surname"`
	Avatar         string   `json:"avatar"`
	ID             string   `json:"id"`
	IsAdmin        bool     `json:"isAdmin"`
	IsOwner        bool     `json:"isOwner"`
	IsSpecialGuest bool     `json:"isSpecialGuest"`
	Badges         []string `json:"badges"`

	Reaction string `json:"reaction"`
	IsLocal  bool   `json:"isLocal"`
}

type StateResponseMessage struct {
	Room                   map[string]Participant `json:"room"`
	Popup                  []ShortParticipant     `json:"popup"`
	ListenersCount         int                    `json:"listenersCount"`
	Current                Participant            `json:"current"`
	Admins                 []string               `json:"admins"`
	Hands                  []string               `json:"hands"`
	OwnerID                string                 `json:"owner"`
	OnlineAdmins           []string               `json:"onlineAdmins"`
	HandsAllowed           bool                   `json:"handsAllowed"`
	AbsoluteSpeakerPresent bool                   `json:"absoluteSpeakerPresent"`
}

type Participant struct {
	Name              string   `json:"name"`
	Surname           string   `json:"surname"`
	Avatar            string   `json:"avatar"`
	X                 float64  `json:"x"`
	Y                 float64  `json:"y"`
	IsSpeaker         bool     `json:"isSpeaker"`
	AudioLevel        int64    `json:"audioLevel"`
	Expired           bool     `json:"expired"`
	InRadar           bool     `json:"inRadar"`
	Volume            float64  `json:"volume"`
	Jitsi             bool     `json:"jitsi"`
	Video             bool     `json:"video"`
	Audio             bool     `json:"audio"`
	PhoneCall         bool     `json:"phoneCall"`
	Mode              string   `json:"mode"`
	ID                string   `json:"id"`
	IsAdmin           bool     `json:"isAdmin"`
	IsOwner           bool     `json:"isOwner"`
	IsSpecialGuest    bool     `json:"isSpecialGuest"`
	IsAbsoluteSpeaker bool     `json:"isAbsoluteSpeaker"`
	Badges            []string `json:"badges"`
}
