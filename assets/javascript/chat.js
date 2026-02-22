const messageInput = document.getElementById("messageInput")
const sendBtn = document.getElementById("sendBtn")
const chatMessages = document.getElementById("chatMessages")

let chatWs = null
const STICKER_PREFIX = "[[sticker:"
const STICKER_SUFFIX = "]]"
const STICKER_COOLDOWN_MS = 20_000
const STICKER_SOUNDS = {
	dacha: "/sounds/Dacha.MP3",
}
let activeStickerAudio = null
let stickerCooldownUntil = 0

function sendMessage() {
	const text = messageInput.value.trim()
	if (!text || !chatWs || chatWs.readyState !== WebSocket.OPEN) {
		return
	}
	chatWs.send(text)
	messageInput.value = ""
}

if (sendBtn) {
	sendBtn.addEventListener("click", sendMessage)
}

if (messageInput) {
	messageInput.addEventListener("keydown", (event) => {
		if (event.key === "Enter") {
			event.preventDefault()
			sendMessage()
		}
	})
}

window.addEventListener("load", () => {
	if (chatMessages) {
		chatMessages.scrollTop = chatMessages.scrollHeight
	}
})

function escapeHtml(value) {
	return String(value)
		.replaceAll("&", "&amp;")
		.replaceAll("<", "&lt;")
		.replaceAll(">", "&gt;")
		.replaceAll('"', "&quot;")
		.replaceAll("'", "&#39;")
}

function currentTime() {
	const date = new Date()
	const hour = String(date.getHours()).padStart(2, "0")
	const minute = String(date.getMinutes()).padStart(2, "0")
	return `${hour}:${minute}`
}

function renderMessage(item) {
	if (!chatMessages) return

	const author = (item.author || "anonymous").trim()
	const text = (item.text || "").trim()
	const time = (item.time || currentTime()).trim()
	if (!text) return

	const avatar = escapeHtml(author.charAt(0).toUpperCase() || "A")
	const messageDiv = document.createElement("div")
	messageDiv.className = "chat-message"
	messageDiv.innerHTML = `
    <div class="message-avatar">${avatar}</div>
    <div class="message-content">
      <div class="message-header">
        <span class="message-author">${escapeHtml(author)}</span>
        <span class="message-time">${escapeHtml(time)}</span>
      </div>
      <div class="message-text">${escapeHtml(text)}</div>
    </div>
  `

	chatMessages.appendChild(messageDiv)
	chatMessages.scrollTop = chatMessages.scrollHeight
}

function parseWireMessage(raw) {
	try {
		const parsed = JSON.parse(raw)
		if (parsed && typeof parsed === "object") {
			return {
				author: parsed.author || "anonymous",
				time: parsed.time || currentTime(),
				text: parsed.text || "",
			}
		}
	} catch (_) {}

	// Backward compatibility for plain text messages.
	return {
		author: "anonymous",
		time: currentTime(),
		text: raw,
	}
}

function encodeStickerMessage(stickerId) {
	return `${STICKER_PREFIX}${stickerId}${STICKER_SUFFIX}`
}

function parseStickerMessage(text) {
	const value = String(text || "").trim()
	if (!value.startsWith(STICKER_PREFIX) || !value.endsWith(STICKER_SUFFIX)) {
		return null
	}
	const start = STICKER_PREFIX.length
	const end = value.length - STICKER_SUFFIX.length
	const stickerId = value.slice(start, end).trim()
	return stickerId || null
}

function playStickerSound(stickerId) {
	const url = STICKER_SOUNDS[stickerId]
	if (!url) return

	try {
		if (activeStickerAudio) {
			activeStickerAudio.pause()
			activeStickerAudio.currentTime = 0
		}
		activeStickerAudio = new Audio(url)
		activeStickerAudio.play().catch(() => {})
	} catch (_) {}
}

window.sendChatSticker = function (stickerId) {
	if (!chatWs || chatWs.readyState !== WebSocket.OPEN) {
		return false
	}
	chatWs.send(encodeStickerMessage(stickerId))
	return true
}

function bindStickerPanel() {
	const stickerButtons = document.querySelectorAll(".js-sticker")
	if (!stickerButtons.length) return

	const setStickerButtonsDisabled = (disabled) => {
		stickerButtons.forEach((button) => {
			button.disabled = disabled
		})
	}

	stickerButtons.forEach((button) => {
		button.addEventListener("click", () => {
			const now = Date.now()
			if (now < stickerCooldownUntil) {
				return
			}

			const stickerId = button.dataset.sticker
			if (!stickerId) return
			const sent = window.sendChatSticker(stickerId)
			if (sent) {
				stickerCooldownUntil = now + STICKER_COOLDOWN_MS
				setStickerButtonsDisabled(true)
				setTimeout(() => {
					if (Date.now() >= stickerCooldownUntil) {
						setStickerButtonsDisabled(false)
					}
				}, STICKER_COOLDOWN_MS)
				return
			}

			console.log("chat ws is not connected yet")
		})
	})
}

function connectChat() {
	chatWs = new WebSocket(ChatWebsocketAddr)

	chatWs.onmessage = function (evt) {
		const messages = String(evt.data).split("\n")
		for (const raw of messages) {
			if (!raw) continue
			const parsed = parseWireMessage(raw)
			const stickerId = parseStickerMessage(parsed.text)
			if (stickerId) {
				playStickerSound(stickerId)
				continue
			}
			renderMessage(parsed)
		}
	}

	chatWs.onerror = function (evt) {
		console.log("chat ws error:", evt)
	}
}

connectChat()
bindStickerPanel()
