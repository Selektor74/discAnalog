function connectStream() {
	document.getElementById('peers').style.display = 'block'
	document.getElementById('chat').style.display = 'flex'
	const TURN_USERNAME = (typeof TurnUsername === "string" ? TurnUsername.trim() : "")
	const TURN_PASSWORD = (typeof TurnPassword === "string" ? TurnPassword : "")
	const TURN_HOST = (typeof TurnHost === "string" ? TurnHost.trim() : "")
	const TURN_PORT = (typeof TurnPort === "string" && TurnPort.trim() ? TurnPort.trim() : "3478")
	const iceServers = [{ urls: "stun:stun.l.google.com:19302" }]
	if (TURN_HOST && TURN_USERNAME && TURN_PASSWORD) {
		iceServers.push({
			urls: [
				`turn:${TURN_HOST}:${TURN_PORT}?transport=udp`,
				`turn:${TURN_HOST}:${TURN_PORT}?transport=tcp`,
			],
			username: TURN_USERNAME,
			credential: TURN_PASSWORD,
		})
	}

	let pc = new RTCPeerConnection({
		iceTransportPolicy: "relay",
		iceServers,
	})
	pc.oniceconnectionstatechange = () => {
		console.log("stream iceConnectionState:", pc.iceConnectionState)
	}
	pc.onconnectionstatechange = () => {
		console.log("stream connectionState:", pc.connectionState)
	}
	pc.onicecandidateerror = (e) => {
		console.log("stream icecandidateerror:", e?.url, e?.errorCode, e?.errorText)
	}

	pc.ontrack = function (event) {

		col = document.createElement("div")
		col.className = "column is-6 peer"
		let el = document.createElement(event.track.kind)
		el.srcObject = event.streams[0]
		el.setAttribute("controls", "true")
		el.setAttribute("autoplay", "true")
		el.setAttribute("playsinline", "true")
		let playAttempt = setInterval(() => {
			el.play()
				.then(() => {
					clearInterval(playAttempt);
				})
				.catch(error => {
					console.log('unable to play the video, user has not interacted yet');
				});
		}, 3000);

		col.appendChild(el)
		document.getElementById('noonestream').style.display = 'none'
		document.getElementById('nocon').style.display = 'none'
		document.getElementById('videos').appendChild(col)

		event.track.onmute = function (event) {
			el.play()
		}

		event.streams[0].onremovetrack = ({
			track
		}) => {
			if (el.parentNode) {
				el.parentNode.remove()
			}
			if (document.getElementById('videos').childElementCount <= 2) {
				document.getElementById('noonestream').style.display = 'flex'
			}
		}
	}

	let ws = new WebSocket(StreamWebsocketAddr)
	pc.onicecandidate = e => {
		if (!e.candidate) {
			return
		}

		ws.send(JSON.stringify({
			event: 'candidate',
			data: JSON.stringify(e.candidate)
		}))
	}

	ws.addEventListener('error', function (event) {
		console.log('error: ', event)
	})

	ws.onclose = function (evt) {
		console.log("websocket has closed")
		pc.close();
		pc = null;
		pr = document.getElementById('videos')
		while (pr.childElementCount > 2) {
			pr.lastChild.remove()
		}
		document.getElementById('noonestream').style.display = 'none'
		document.getElementById('nocon').style.display = 'flex'
		setTimeout(function () {
			connectStream();
		}, 1000);
	}

	ws.onmessage = function (evt) {
		let msg = JSON.parse(evt.data)
		if (!msg) {
			return console.log('failed to parse msg')
		}

		switch (msg.event) {
			case 'offer':
				let offer = JSON.parse(msg.data)
				if (!offer) {
					return console.log('failed to parse answer')
				}
				pc.setRemoteDescription(offer)
				pc.createAnswer().then(answer => {
					pc.setLocalDescription(answer)
					ws.send(JSON.stringify({
						event: 'answer',
						data: JSON.stringify(answer)
					}))
				})
				return

			case 'candidate':
				let candidate = JSON.parse(msg.data)
				if (!candidate) {
					return console.log('failed to parse candidate')
				}

				pc.addIceCandidate(candidate)
		}
	}

	ws.onerror = function (evt) {
		console.log("error: " + evt.data)
	}
}

connectStream();
