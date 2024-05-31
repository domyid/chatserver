package wrtc

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

var peers = make(map[*websocket.Conn]*webrtc.PeerConnection)

func RunWebRTCSocket(c *websocket.Conn) {
	defer func() {
		if peerConnection, ok := peers[c]; ok {
			peerConnection.Close()
			delete(peers, c)
		}
		c.Close()
	}()

	config := webrtc.Configuration{}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Println("Failed to create peer connection:", err)
		return
	}

	peers[c] = peerConnection

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		candidateJSON, _ := json.Marshal(candidate.ToJSON())
		c.WriteMessage(websocket.TextMessage, candidateJSON)
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		// Handle incoming tracks
	})

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			return
		}

		var signal map[string]interface{}
		err = json.Unmarshal(message, &signal)
		if err != nil {
			fmt.Println("Error unmarshalling message:", err)
			continue
		}

		handleSignal(peerConnection, signal)
	}
}

func handleSignal(peerConnection *webrtc.PeerConnection, signal map[string]interface{}) {
	if sdp, ok := signal["sdp"]; ok {
		var session webrtc.SessionDescription
		if err := json.Unmarshal([]byte(sdp.(string)), &session); err != nil {
			fmt.Println("Error unmarshalling SDP:", err)
			return
		}

		if session.Type == webrtc.SDPTypeOffer {
			if err := peerConnection.SetRemoteDescription(session); err != nil {
				fmt.Println("Error setting remote description:", err)
				return
			}

			answer, err := peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Println("Error creating answer:", err)
				return
			}

			if err := peerConnection.SetLocalDescription(answer); err != nil {
				fmt.Println("Error setting local description:", err)
				return
			}
		} else if session.Type == webrtc.SDPTypeAnswer {
			if err := peerConnection.SetRemoteDescription(session); err != nil {
				fmt.Println("Error setting remote description:", err)
				return
			}
		}
	} else if candidate, ok := signal["candidate"]; ok {
		var iceCandidate webrtc.ICECandidateInit
		if err := json.Unmarshal([]byte(candidate.(string)), &iceCandidate); err != nil {
			fmt.Println("Error unmarshalling ICE candidate:", err)
			return
		}

		if err := peerConnection.AddICECandidate(iceCandidate); err != nil {
			fmt.Println("Error adding ICE candidate:", err)
			return
		}
	}
}
