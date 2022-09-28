package datatrack

type RequestMessage interface {
	typ() string
}

type RegisterRequestMessage struct {
	SID         string `json:"sid"`
	Password    string `json:"password"`
	AccessToken string `json:"accessToken"`
	Version     int    `json:"version"`
}

func (msg *RegisterRequestMessage) typ() string {
	return "register"
}

type PathRequestMessage struct {
	CurrentX float64 `json:"currentX"`
	CurrentY float64 `json:"currentY"`
	NextX    float64 `json:"nextX"`
	NextY    float64 `json:"nextY"`
	Type     string  `json:"type"`
}

func (msg *PathRequestMessage) typ() string {
	return "getPath"
}

type StateRequestMessage struct {
	Video     bool `json:"video"`
	Audio     bool `json:"audio"`
	PhoneCall bool `json:"phoneCall"`
	Jitsi     bool `json:"jitsi"`
}

func (msg *StateRequestMessage) typ() string {
	return "userState"
}

type AudioLevelRequestMessage struct {
	Value int `json:"value"`
}

func (msg *AudioLevelRequestMessage) typ() string {
	return "audioLevel"
}
