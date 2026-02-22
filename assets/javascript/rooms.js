const roomsListContainer = document.querySelector(".rooms-list")
const createRoomBtn = document.getElementById("createRoomBtn")
const roomNameInput = document.getElementById("roomNameInput")
const createRoomError = document.getElementById("createRoomError")

function renderRooms() {
	if (!roomsListContainer) return

	const rooms = Array.isArray(RoomsList) ? RoomsList : []
	roomsListContainer.innerHTML = ""

	if (rooms.length === 0) {
		const emptyCard = document.createElement("div")
		emptyCard.classList.add("room-card")
		emptyCard.innerHTML = `
      <div>
        <h3 class="room-title">Пока нет комнат</h3>
        <p class="room-info">Создай первую комнату и пригласи друзей.</p>
      </div>
    `
		roomsListContainer.appendChild(emptyCard)
		return
	}

	rooms.forEach((room) => {
		const card = document.createElement("div")
		card.classList.add("room-card")
		card.innerHTML = `
      <div>
        <h3 class="room-title">${room.Name}</h3>
      </div>
      <a href="/room/${room.UUID}" class="join-btn">Присоединиться</a>
    `
		roomsListContainer.appendChild(card)
	})
}

async function createRoom() {
	const name = (roomNameInput?.value || "").trim()
	if (createRoomError) createRoomError.textContent = ""

	if (!createRoomBtn) return
	createRoomBtn.disabled = true
	try {
		const response = await fetch("/room/create", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			credentials: "same-origin",
			body: JSON.stringify({ name }),
		})

		if (!response.ok) {
			let message = "Не удалось создать комнату."
			try {
				const data = await response.json()
				if (data && data.error) message = data.error
			} catch (_) {}
			if (createRoomError) createRoomError.textContent = message
			return
		}

		const data = await response.json()
		if (!data || !data.room_uuid) {
			if (createRoomError) createRoomError.textContent = "Сервер вернул некорректный ответ."
			return
		}

		window.location.href = `/room/${data.room_uuid}`
	} catch (_) {
		if (createRoomError) createRoomError.textContent = "Ошибка сети. Попробуйте еще раз."
	} finally {
		createRoomBtn.disabled = false
	}
}

if (createRoomBtn) {
	createRoomBtn.addEventListener("click", createRoom)
}

if (roomNameInput) {
	roomNameInput.addEventListener("keydown", (event) => {
		if (event.key === "Enter") {
			event.preventDefault()
			createRoom()
		}
	})
}

renderRooms()
