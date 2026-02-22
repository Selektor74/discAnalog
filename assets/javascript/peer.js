// Состояние кнопок
let isMuted = false
let isVideoOff = false
let localCombinedStream = null

// Элементы
const muteBtn = document.getElementById("muteBtn")
const videoBtn = document.getElementById("videoBtn")
const shareBtn = document.getElementById("shareBtn")
const leaveBtn = document.getElementById("leaveBtn")
const localVideoEl = document.getElementById("localVideo")
const localVideoPlaceholderEl = document.getElementById("localVideoPlaceholder")
const TURN_USERNAME = (typeof TurnUsername === "string" ? TurnUsername.trim() : "")
const TURN_PASSWORD = (typeof TurnPassword === "string" ? TurnPassword : "")
const TURN_HOST = (typeof TurnHost === "string" ? TurnHost.trim() : "")
const TURN_PORT = (typeof TurnPort === "string" && TurnPort.trim() ? TurnPort.trim() : "3478")

function applyMuteState() {
	if (!localCombinedStream) return
	localCombinedStream.getAudioTracks().forEach(track => {
		track.enabled = !isMuted
	})
}

function applyVideoState() {
	if (!localCombinedStream) return
	localCombinedStream.getVideoTracks().forEach(track => {
		track.enabled = !isVideoOff
	})
	updateLocalVideoVisibility()
}

function updateLocalVideoVisibility() {
	if (!localVideoEl || !localVideoPlaceholderEl) return
	if (isVideoOff) {
		localVideoEl.style.display = "none"
		localVideoPlaceholderEl.style.display = "flex"
		return
	}
	localVideoEl.style.display = ""
	localVideoPlaceholderEl.style.display = "none"
}

// Обработчики кнопок управления
muteBtn.addEventListener("click", function () {
	isMuted = !isMuted
	applyMuteState()
	this.classList.toggle("muted", isMuted)
	this.setAttribute("aria-label", isMuted ? "Включить микрофон" : "Выключить микрофон")
})

videoBtn.addEventListener("click", function () {
	isVideoOff = !isVideoOff
	applyVideoState()
	this.classList.toggle("video-off", isVideoOff)
	this.setAttribute("aria-label", isVideoOff ? "Включить видео" : "Выключить видео")
})

shareBtn.addEventListener("click", () => {
	alert('Функция "Поделиться экраном" будет доступна в полной версии')
})

leaveBtn.addEventListener("click", () => {
	if (confirm("Вы уверены, что хотите покинуть встречу?")) {
		window.location.href = "/rooms"
	}
})

function copyToClipboard(text) {
	if (window.clipboardData && window.clipboardData.setData) {
		clipboardData.setData("Text", text);
		return Swal.fire({
			position: 'top-end',
			text: "Copied",
			showConfirmButton: false,
			timer: 1000,
			width: '150px'
		})
	} else if (document.queryCommandSupported && document.queryCommandSupported("copy")) {
		var textarea = document.createElement("textarea");
		textarea.textContent = text;
		textarea.style.position = "fixed";
		document.body.appendChild(textarea);
		textarea.select();
		try {
			document.execCommand("copy");
			return Swal.fire({
				position: 'top-end',
				text: "Copied",
				showConfirmButton: false,
				timer: 1000,
				width: '150px'
			})
		} catch (ex) {
			console.warn("Copy to clipboard failed.", ex);
			return false;
		} finally {
			document.body.removeChild(textarea);
		}
	}
}

document.addEventListener('DOMContentLoaded', () => {
	(document.querySelectorAll('.notification .delete') || []).forEach(($delete) => {
		const $notification = $delete.parentNode;

		$delete.addEventListener('click', () => {
			$notification.style.display = 'none'
		});
	});
});

function connect(stream) {
	document.getElementById('peers').style.display = 'block'
	document.getElementById('chat').style.display = 'flex'
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

	const pc = new RTCPeerConnection({
		iceTransportPolicy: "relay",
		iceServers,
	})
	pc.oniceconnectionstatechange = () => {
		console.log("peer iceConnectionState:", pc.iceConnectionState)
	}
	pc.onconnectionstatechange = () => {
		console.log("peer connectionState:", pc.connectionState)
	}
	pc.onicecandidateerror = (e) => {
		console.log("peer icecandidateerror:", e?.url, e?.errorCode, e?.errorText)
	}

	const remoteTiles = new Map()
	function ensurePlaying(el) {
		if (!el) return
		const tryPlay = () => el.play().catch(() => { })
		el.onloadedmetadata = () => tryPlay()
		tryPlay()
		setTimeout(tryPlay, 300)
		setTimeout(tryPlay, 1200)
	}

	function ensureRemoteTile(stream) {
		const existing = remoteTiles.get(stream.id)
		if (existing) return existing

		const tileDiv = document.createElement("div")
		tileDiv.className = "participant-tile"

		const infoDiv = document.createElement("div")
		infoDiv.className = "participant-info"

		const userIcon = document.createElement("div")
		userIcon.className = "participant-video"
		userIcon.textContent = "👤"

		const nameSpan = document.createElement("span")
		nameSpan.className = "participant-name"

		const micSpan = document.createElement("span")
		micSpan.className = "mic-indicator mic-on"
		micSpan.textContent = "🎤"

		tileDiv.appendChild(userIcon)
		infoDiv.appendChild(nameSpan)
		infoDiv.appendChild(micSpan)
		tileDiv.appendChild(infoDiv)
		document.getElementById('videos').appendChild(tileDiv)
		document.getElementById('noone').style.display = 'none'
		document.getElementById('nocon').style.display = 'none'

		const entry = { tileDiv, infoDiv, userIcon, videoEl: null, audioEl: null, stream }
		remoteTiles.set(stream.id, entry)
		return entry
	}

	function removeRemoteTile(streamId) {
		const entry = remoteTiles.get(streamId)
		if (!entry) return
		if (entry.tileDiv && entry.tileDiv.parentNode) {
			entry.tileDiv.remove()
		}
		remoteTiles.delete(streamId)
		if (remoteTiles.size === 0) {
			document.getElementById('noone').style.display = 'flex'
		}
	}

	function refreshTileByStream(stream, entry) {
		const hasLiveVideo = stream.getVideoTracks().some(t => t.readyState === "live")
		const hasLiveAudio = stream.getAudioTracks().some(t => t.readyState === "live")

		if (!hasLiveVideo) {
			if (entry.videoEl && entry.videoEl.parentNode) entry.videoEl.remove()
			entry.videoEl = null
			if (!entry.userIcon.parentNode) entry.tileDiv.insertBefore(entry.userIcon, entry.infoDiv)
		}
		if (!hasLiveAudio) {
			if (entry.audioEl && entry.audioEl.parentNode) entry.audioEl.remove()
			entry.audioEl = null
		}
		if (!hasLiveVideo && !hasLiveAudio) {
			removeRemoteTile(stream.id)
		}
	}

	pc.ontrack = function (event) {
		const stream = event.streams && event.streams[0]
		if (!stream) return

		console.log("ontrack:", event.track.kind, stream.id, event.track.id)
		const entry = ensureRemoteTile(stream)

		if (event.track.kind === "video" && !entry.videoEl) {
			const videoEl = document.createElement("video")
			videoEl.autoplay = true
			videoEl.playsInline = true
			videoEl.className = "participant-video"
			videoEl.srcObject = stream
			if (entry.userIcon.parentNode) entry.userIcon.remove()
			entry.tileDiv.insertBefore(videoEl, entry.infoDiv)
			entry.videoEl = videoEl
			ensurePlaying(videoEl)
		}
		if (event.track.kind === "audio" && !entry.audioEl) {
			const audioEl = document.createElement("audio")
			audioEl.autoplay = true
			audioEl.hidden = true
			audioEl.srcObject = stream
			entry.tileDiv.appendChild(audioEl)
			entry.audioEl = audioEl
			ensurePlaying(audioEl)
		}

		event.track.onmute = function () {
			if (entry.videoEl) entry.videoEl.play().catch(() => { })
			if (entry.audioEl) entry.audioEl.play().catch(() => { })
		}
		event.track.onended = function () {
			refreshTileByStream(stream, entry)
		}
		stream.onremovetrack = function () {
			refreshTileByStream(stream, entry)
		}
	}

	stream.getTracks().forEach(track => pc.addTrack(track, stream))
	const VIDEO_MAX_BITRATE_BPS = 450_000
	pc.getSenders().forEach((sender) => {
		if (!sender.track || sender.track.kind !== "video") return
		const params = sender.getParameters()
		if (!params.encodings || params.encodings.length === 0) {
			params.encodings = [{}]
		}
		params.encodings[0].maxBitrate = VIDEO_MAX_BITRATE_BPS
		sender.setParameters(params).catch(() => { })
	})

	const ws = new WebSocket(RoomWebsocketAddr)
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
		pr = document.getElementById('videos')
		document.getElementById('noone').style.display = 'none'
		document.getElementById('nocon').style.display = 'flex'
		setTimeout(function () {
			connect(stream);
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


async function getAudioStream() {
	try {
		const audioStream = await navigator.mediaDevices.getUserMedia({
			audio: {
				sampleSize: 16,
				channelCount: 2,
				echoCancellation: true
			}
		})
		if (audioStream && audioStream.getAudioTracks().length > 0) {
			return audioStream
		} else {
			console.log("Микрофон не найден")
			return null
		}
	} catch (err) {
		console.log("Ошибка микрофона", err)
		return null

	}
}

async function getVideoStream() {
	try {
		const videoStream = await navigator.mediaDevices.getUserMedia({
			video: {
				width: {
					ideal: 854,
					max: 854
				},
				height: {
					ideal: 480,
					max: 480
				},
				frameRate: {
					ideal: 24,
					max: 24
				},
			}
		})
		if (videoStream && videoStream.getVideoTracks().length > 0) {
			return videoStream
		} else {
			console.log("Камера не найдена")
			return null
		}
	} catch (err) {
		console.log("Ошибка камеры", err)
		return null

	}
}

async function getCombinedStream() {

	audioStream = await getAudioStream()
	videoStream = await getVideoStream()


	const combinedStream = new MediaStream()

	if (audioStream != null) {
		audioStream.getTracks().forEach(track => combinedStream.addTrack(track))

	}

	if (videoStream != null) {
		videoStream.getTracks().forEach(track => combinedStream.addTrack(track))

	}




	return combinedStream

}

getCombinedStream().then(stream => {
	localCombinedStream = stream
	applyMuteState()
	applyVideoState()
	if (localVideoEl) {
		localVideoEl.srcObject = stream
	}
	updateLocalVideoVisibility()
	connect(stream)
}).catch(err => console.log(err))




