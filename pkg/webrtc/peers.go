package webrtc

import (
	"SelektorDisc/pkg/chat"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

var (
	RoomsLock sync.RWMutex
	Rooms     map[string]*RoomRTC
	Streams   map[string]*RoomRTC
)

var (
	turnConfig = buildTURNConfig()
)

func buildTURNConfig() webrtc.Configuration {
	host := firstNonEmptyEnv("TURN_HOST", "TURN_PUBLIC_IP")
	port := os.Getenv("TURN_PORT")
	if strings.TrimSpace(port) == "" {
		port = "3478"
	}

	username := os.Getenv("TURN_USERNAME")
	password := os.Getenv("TURN_PASSWORD")
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		fallbackUser, fallbackPass := parseTURNUsers(os.Getenv("TURN_USERS"))
		if strings.TrimSpace(username) == "" {
			username = fallbackUser
		}
		if strings.TrimSpace(password) == "" {
			password = fallbackPass
		}
	}

	if strings.TrimSpace(host) == "" || strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		log.Println("webrtc: TURN is not fully configured for backend; using default ICE config")
		return webrtc.Configuration{}
	}

	// In production backend should prefer relay candidates to avoid private container IPs in SDP.
	policy := webrtc.ICETransportPolicyAll
	if IsProduction() || strings.EqualFold(strings.TrimSpace(os.Getenv("TURN_FORCE_RELAY")), "true") {
		policy = webrtc.ICETransportPolicyRelay
	}

	return webrtc.Configuration{
		ICETransportPolicy: policy,
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{
					fmt.Sprintf("turn:%s:%s?transport=udp", host, port),
					fmt.Sprintf("turn:%s:%s?transport=tcp", host, port),
				},
				Username:       username,
				Credential:     password,
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func parseTURNUsers(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}

	// Expected format: user=password[,user2=password2]
	firstEntry := strings.Split(raw, ",")[0]
	parts := strings.SplitN(strings.TrimSpace(firstEntry), "=", 2)
	if len(parts) != 2 {
		return "", ""
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

type RoomRTC struct {
	RoomID string
	Name   string
	Peers *Peers
	Hub   *chat.Hub
}

type Peers struct {
	ListLock    sync.RWMutex
	Connections []PeerConnectionState
	TrackLocals map[string]*webrtc.TrackLocalStaticRTP
	TrackOwners map[string]*webrtc.PeerConnection
}

func (p *Peers) CountConnections() int {
	p.ListLock.RLock()
	defer p.ListLock.RUnlock()
	return len(p.Connections)
}

type PeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Websocket      *ThreadSafeWriter
}

type ThreadSafeWriter struct {
	Conn  *websocket.Conn
	Mutex sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.Conn.WriteJSON(v)
}

func trackLocalKey(t *webrtc.TrackRemote) string {
	// Remote IDs like "audio"/"video" are often duplicated across peers.
	// Include stream ID and SSRC to keep local tracks unique per publisher.
	return fmt.Sprintf("%s|%s|%d", t.StreamID(), t.ID(), t.SSRC())
}

func (p *Peers) AddTrack(t *webrtc.TrackRemote, owner *webrtc.PeerConnection) *webrtc.TrackLocalStaticRTP {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.SignalPeerConnections()
	}()

	localTrackID := trackLocalKey(t)
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, localTrackID, t.StreamID())

	if err != nil {
		log.Println(err.Error())
		return nil
	}
	p.TrackLocals[localTrackID] = trackLocal
	if p.TrackOwners == nil {
		p.TrackOwners = make(map[string]*webrtc.PeerConnection)
	}
	p.TrackOwners[localTrackID] = owner
	return trackLocal
}

func (p *Peers) RemoveTrack(t *webrtc.TrackLocalStaticRTP) {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.SignalPeerConnections()
	}()
	delete(p.TrackLocals, t.ID())
	delete(p.TrackOwners, t.ID())

}

func (p *Peers) SignalPeerConnections() {
	p.ListLock.Lock()
	defer func() {
		p.ListLock.Unlock()
		p.DispatchKeyFrame()
	}()

	attempSync := func() (tryAgain bool) {
		for i := range p.Connections {
			if p.Connections[i].PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				p.Connections = append(p.Connections[:i], p.Connections[i+1:]...)
				log.Println("a", p.Connections)
				return true
			}
			existingSenders := map[string]bool{}
			for _, sender := range p.Connections[i].PeerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				if _, ok := p.TrackLocals[sender.Track().ID()]; !ok {
					if err := p.Connections[i].PeerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}
			for _, reciever := range p.Connections[i].PeerConnection.GetReceivers() {
				if reciever.Track() == nil {
					continue
				}
				existingSenders[reciever.Track().ID()] = true
			}
			for trackId := range p.TrackLocals {
				if _, ok := existingSenders[trackId]; !ok {
					if p.TrackOwners[trackId] == p.Connections[i].PeerConnection {
						continue
					}
					if _, err := p.Connections[i].PeerConnection.AddTrack(p.TrackLocals[trackId]); err != nil {
						return true
					}
				}
			}
			offer, err := p.Connections[i].PeerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = p.Connections[i].PeerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			if err = p.Connections[i].Websocket.WriteJSON(&websocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}
		return
	}

	for syncAttemp := 0; ; syncAttemp++ {
		if syncAttemp == 25 {
			go func() {
				time.Sleep(time.Second * 3)
				p.SignalPeerConnections()
			}()
			return
		}
		if !attempSync() {
			break
		}
	}

}

func (p *Peers) DispatchKeyFrame() {

	p.ListLock.Lock()
	defer p.ListLock.Unlock()

	for i := range p.Connections {
		for _, reciever := range p.Connections[i].PeerConnection.GetReceivers() {
			if reciever.Track() == nil {
				continue
			}

			_ = p.Connections[i].PeerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(reciever.Track().SSRC()),
				},
			})
		}
	}
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
