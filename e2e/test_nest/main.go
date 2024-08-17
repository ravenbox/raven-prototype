//go:build !js
// +build !js

package main

// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// reflect demonstrates how with one PeerConnection you can send video to Pion and have the packets sent back

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ravenbox/raven-prototype/pkg/nest"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/webrtc/v4"
)

// nolint:gocognit
func main() {
	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	i := &interceptor.Registry{}

	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}

	intervalPliFactory, err := intervalpli.NewReceiverInterceptor()
	if err != nil {
		panic(err)
	}
	i.Add(intervalPliFactory)

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	ary := make([]*webrtc.PeerConnection, 2)
	for i := range ary {
		fmt.Println("Connection", i)
		peerConnection, err := api.NewPeerConnection(config)
		if err != nil {
			panic(err)
		}
		defer func() {
			if cErr := peerConnection.Close(); cErr != nil {
				fmt.Printf("cannot close peerConnection: %v\n", cErr)
			}
		}()

		offer := webrtc.SessionDescription{}
		decode(readUntilNewline(), &offer)

		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			panic(err)
		}

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			panic(err)
		}
		//
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
		if err = peerConnection.SetLocalDescription(answer); err != nil {
			panic(err)
		}
		<-gatherComplete
		fmt.Println(encode(peerConnection.LocalDescription()))
		ary[i] = peerConnection
	}

	nst := nest.NewNest()
	for i := range ary {
		id := fmt.Sprintf("qolam-%d", i)
		fmt.Println("Registering Conn", id)
		nst.RegisterPeer(id, ary[i])
		if i != 0 {
			err := nst.SubscribeToTrack(id, "qolam-0#video/VP8")
			if err != nil {
				panic(err)
			}
		}
	}

	fmt.Println("DONE")
	// Block forever
	select {}
}

func readUntilNewline() (in string) {
	fmt.Scanf("%s", &in)
	return
}

// JSON encode + base64 a SessionDescription
func encode(obj *webrtc.SessionDescription) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode a base64 and unmarshal JSON into a SessionDescription
func decode(in string, obj *webrtc.SessionDescription) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, obj); err != nil {
		panic(err)
	}
}
