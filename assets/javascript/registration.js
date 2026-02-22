const registerForm = document.getElementById("registerForm")
const usernameInput = document.getElementById("username")
const passwordInput = document.getElementById("password")
const confirmPasswordInput = document.getElementById("confirm-password")
const registerBtn = document.getElementById("registerBtn")
const registerError = document.getElementById("registerError")
const minPasswordLength = 8

if (registerForm) {
	registerForm.addEventListener("submit", async (event) => {
		event.preventDefault()

		const username = usernameInput.value.trim()
		const password = passwordInput.value.trim()
		const confirmPassword = confirmPasswordInput.value.trim()

		registerError.textContent = ""
		if (!username || !password) {
			registerError.textContent = "Введите имя пользователя и пароль."
			return
		}
		if (password.length < minPasswordLength) {
			registerError.textContent = `Пароль должен содержать минимум ${minPasswordLength} символов.`
			return
		}
		if (password !== confirmPassword) {
			registerError.textContent = "Пароли не совпадают."
			return
		}

		registerBtn.disabled = true
		try {
			const response = await fetch("/auth/register", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				credentials: "same-origin",
				body: JSON.stringify({ username, password }),
			})

			if (!response.ok) {
				let message = `Ошибка регистрации (HTTP ${response.status}).`
				try {
					const data = await response.json()
					if (data && data.error) {
						message = data.error
					}
				} catch (_) {
					const text = await response.text()
					if (text) {
						message = text
					}
				}
				registerError.textContent = message
				return
			}

			window.location.href = "/rooms"
		} catch (error) {
			console.error("register request failed:", error)
			registerError.textContent = "Ошибка сети. Сервер недоступен или не отвечает."
		} finally {
			registerBtn.disabled = false
		}
	})
}
