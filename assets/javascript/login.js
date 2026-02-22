const loginForm = document.getElementById("loginForm")
const usernameInput = document.getElementById("login-username")
const passwordInput = document.getElementById("login-password")
const loginBtn = document.getElementById("loginBtn")
const loginError = document.getElementById("loginError")

if (loginForm) {
	loginForm.addEventListener("submit", async (event) => {
		event.preventDefault()

		const username = usernameInput.value.trim()
		const password = passwordInput.value.trim()

		loginError.textContent = ""
		if (!username || !password) {
			loginError.textContent = "Введите логин и пароль."
			return
		}

		loginBtn.disabled = true
		try {
			const response = await fetch("/auth/login", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				credentials: "same-origin",
				body: JSON.stringify({ username, password }),
			})

			if (!response.ok) {
				let message = "Не удалось войти. Проверьте логин и пароль."
				try {
					const data = await response.json()
					if (data && data.error) {
						message = data.error
					}
				} catch (_) {}
				loginError.textContent = message
				return
			}

			window.location.href = "/rooms"
		} catch (_) {
			loginError.textContent = "Ошибка сети. Попробуйте еще раз."
		} finally {
			loginBtn.disabled = false
		}
	})
}
