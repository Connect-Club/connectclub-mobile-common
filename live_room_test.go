package common

import (
	"fmt"
	"testing"
	"time"
)

type TestLiveRoomDatatrackDelegate struct{}

func (t *TestLiveRoomDatatrackDelegate) OnStateChanged(newState int) {
	fmt.Println("DatatrackDelegate.OnStateChanged", LiveRoomState(newState))
}

func (t *TestLiveRoomDatatrackDelegate) OnMessage(json string) {
	fmt.Println("DatatrackDelegate.OnMessage", json)
}

func (t *TestLiveRoomDatatrackDelegate) OnReaction(json []byte) {
	fmt.Println("DatatrackDelegate.OnReaction", string(json))
}

func (t *TestLiveRoomDatatrackDelegate) OnPath(userId string, x float64, y float64, duration float64) {
	fmt.Println("DatatrackDelegate.OnPath", userId, x, y, duration)
}

func (t *TestLiveRoomDatatrackDelegate) OnNativeState(json []byte) {
	fmt.Println("DatatrackDelegate.OnNativeState", string(json))
}

func (t *TestLiveRoomDatatrackDelegate) OnRadarVolume(json []byte) {
	fmt.Println("DatatrackDelegate.OnRadarVolume", string(json))
}

func (t *TestLiveRoomDatatrackDelegate) OnConnectionState(json []byte) {
	fmt.Println("DatatrackDelegate.OnConnectionState", string(json))
}

func (t *TestLiveRoomDatatrackDelegate) OnPopupUsers(json string) {
	fmt.Println("DatatrackDelegate.OnPopupUsers", json)
}

func (t *TestLiveRoomDatatrackDelegate) OnChangeRoomMode(mode string, isFirstConnection bool) {
	fmt.Println("DatatrackDelegate.OnChangeRoomMode", mode, isFirstConnection)
}

func (t *TestLiveRoomDatatrackDelegate) OnParticipantsVisibilityChanged(json string) {
	fmt.Println("DatatrackDelegate.OnParticipantsVisibilityChanged", json)
}

type TestJvbusterDelegate struct{}

func (t *TestJvbusterDelegate) InitializeRTCPeerConnection(userId string, id string, isMain bool, isSpeaker bool, sdpOffer string) string {
	fmt.Println("JvbusterDelegate.InitializeRTCPeerConnection", userId, id, isMain, isSpeaker, sdpOffer)
	return ""
}

func (t *TestJvbusterDelegate) DestroyPeerConnections() {
	fmt.Println("JvbusterDelegate.DestroyPeerConnections")
}

func (t *TestJvbusterDelegate) SetLocalRTCPeerConnectionDescription(id string, description string) {
	fmt.Println("JvbusterDelegate.SetLocalRTCPeerConnectionDescription", id, description)
}

func (t *TestJvbusterDelegate) SendMessageToDataChannel(id string, message string) {
	fmt.Println("JvbusterDelegate.SendMessageToDataChannel", id, message)
}

func (t *TestJvbusterDelegate) ProcessMetaAdd(json string) {
	fmt.Println("JvbusterDelegate.ProcessMetaAdd", json)
}

func (t *TestJvbusterDelegate) ProcessMetaRemove(json string) {
	fmt.Println("JvbusterDelegate.ProcessMetaRemove", json)
}

func (t *TestJvbusterDelegate) OnJvBusterError(error string) {
	fmt.Println("JvbusterDelegate.OnJvBusterError", error)
}

type testStorage map[string]string

func (s testStorage) GetString(key string) string {
	return s[key]
}

func (s testStorage) SetString(key, value string) {
	s[key] = value
}

func (s testStorage) Delete(key string) {
	delete(s, key)
}

var (
	accessToken   = "MjZjNmVhOTE2OWVjMWU4Zjc5ZTVkNWMzOWQzZWIxY2M0MDY1ODM1ZWJhNTFlNDYzNDkyMTc0MjQyZTFhZTI5Zg"
	jvbusterToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbmRwb2ludCI6IjIxMjEiLCJjb25mZXJlbmNlR2lkIjoiNjIzNDk3NTAxMDQyMCJ9.MCHMp4kIrBxzMmJgluLV4oiQCpAg6KrsjFr6S8XDBCqSLqqLOfskCUECiFiikUqbd2_PV4nUFUrYG6IcGp1d69PH08h5aGIP71oxSFa2PtsnVNufU9Vbazh9sRyM3ZVAROLGUeSwxqCb5xPbQBYNCi9LJZI-n_W6ggFTikmfTI61tgiwco2UoX5sP20OcfTlR2ZsQeJeMCfALuXk0UrFklt6LWm2lLG8QgQoJyKzeCrqbSV_hWcWd23GypkX7B7PcXfBkrl8JOcstXVx1_O7pRe02d6Qqiiq9f6vFheKNbAMCTs1qAtSGd3OlDRaYV1I5LGcqIPTFFU-9jNgywrbZw"
)

func TestConnectToLiveRoom(t *testing.T) {
	//SetStorage(make(testStorage))
	datatrackDelegate := &TestLiveRoomDatatrackDelegate{}
	jvbusterDelegate := &TestJvbusterDelegate{}
	_, err := ConnectToLiveRoom(
		datatrackDelegate, "wss://dt.stage.connect.club/ws", "6234975010420", accessToken, "H3kKIjmCzf5KbqdJ", 1.0, 1.0, 1.0, 1.0,
		jvbusterDelegate, "https://jitsi-proxy.stage.connect.club", jvbusterToken, 1.0, 1.0, 0, 0,
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	time.Sleep(time.Minute)
}
