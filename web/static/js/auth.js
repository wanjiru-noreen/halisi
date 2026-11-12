"use strict";

const errorBox = document.getElementById("error");
function showError(message) {
  errorBox.textContent = message;
  errorBox.hidden = false;
}

async function request(path, data) {
  const response = await fetch(path, {
    method: data === undefined ? "GET" : "POST",
    credentials: "same-origin",
    headers: data === undefined ? {} : { "Content-Type": "application/json" },
    body: data === undefined ? undefined : JSON.stringify(data),
    cache: "no-store",
  });
  const result = response.status === 204 ? {} : await response.json();
  if (!response.ok) {
    const error = new Error(result.error || "Something went wrong. Please try again.");
    error.status = response.status;
    throw error;
  }
  return result;
}

const form = document.getElementById("auth-form");
if (form) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    errorBox.hidden = true;
    const data = Object.fromEntries(new FormData(form));
    if (new TextEncoder().encode(data.password).length > 72) {
      showError("Password must not exceed 72 bytes. Try a shorter passphrase.");
      return;
    }
    const button = form.querySelector('button[type="submit"]');
    button.disabled = true;
    try {
      await request(form.getAttribute("action"), data);
      form.reset();
      window.location.assign(document.body.dataset.page === "register" ? "/login?registered=1" : "/account");
    } catch (error) {
      showError(error instanceof TypeError ? "Connection interrupted. Please try again." : error.message);
      button.disabled = false;
    }
  });
}

if (document.body.dataset.page === "login" && new URLSearchParams(window.location.search).has("registered")) {
  const notice = document.getElementById("notice");
  notice.textContent = "Your account has been created. Log in to continue.";
  notice.hidden = false;
}

if (document.body.dataset.page === "account") {
  request("/api/auth/me").then(({ user }) => {
    document.getElementById("welcome").textContent = `Welcome, ${user.name}.`;
    document.getElementById("account-email").textContent = user.email;
  }).catch((error) => {
    if (error.status === 401) window.location.replace("/login");
    else showError("We couldn't load your account. Please refresh to try again.");
  });
  document.getElementById("logout").addEventListener("click", async (event) => {
    event.currentTarget.disabled = true;
    try {
      await request("/api/auth/logout", {});
      window.location.replace("/login");
    } catch (error) {
      showError(error.message);
      document.getElementById("logout").disabled = false;
    }
  });
}
