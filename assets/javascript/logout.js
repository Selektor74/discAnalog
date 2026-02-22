function logoutAndRedirect(redirectTo) {
	fetch("/auth/logout", {
		method: "POST",
		credentials: "same-origin"
	}).finally(() => {
		window.location.href = redirectTo || "/"
	})
}

document.querySelectorAll(".js-logout").forEach((el) => {
	el.addEventListener("click", (event) => {
		event.preventDefault()
		const redirectTo = el.getAttribute("data-logout-redirect") || "/"
		logoutAndRedirect(redirectTo)
	})
})
