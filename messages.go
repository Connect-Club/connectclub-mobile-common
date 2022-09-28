package common

type UserPoint struct {
	X, Y float64
}

type DeviceRoomUser struct {
	Id                string   `json:"id"`
	Size              float64  `json:"size"`
	IsLocal           bool     `json:"isLocal"`
	HasRadar          bool     `json:"hasRadar"`
	Name              string   `json:"name"`
	Surname           string   `json:"surname"`
	Avatar            string   `json:"avatar"`
	Mode              string   `json:"mode"`
	IsAdmin           bool     `json:"isAdmin"`
	IsExpired         bool     `json:"isExpired"`
	InRadar           bool     `json:"inRadar"`
	Video             bool     `json:"video"`
	Audio             bool     `json:"audio"`
	PhoneCall         bool     `json:"phoneCall"`
	IsSpecialGuest    bool     `json:"isSpecialGuest"`
	IsAbsoluteSpeaker bool     `json:"isAbsoluteSpeaker"`
	Badges            []string `json:"badges"`
}

type DeviceCurrentUser struct {
	IsAdmin           bool   `json:"isAdmin"`
	IsHandRaised      bool   `json:"isHandRaised"`
	Mode              string `json:"mode"`
	IsAbsoluteSpeaker bool   `json:"isAbsoluteSpeaker"`
}

type DeviceState struct {
	Current                *DeviceCurrentUser `json:"current"`
	Room                   []*DeviceRoomUser  `json:"room"`
	ListenersCount         int                `json:"listenersCount"`
	RaisedHandsCount       int                `json:"raisedHandsCount"`
	HandsAllowed           bool               `json:"handsAllowed"`
	AbsoluteSpeakerPresent bool               `json:"absoluteSpeakerPresent"`
}
