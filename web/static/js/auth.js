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
  notice.textContent = "Account created. Check your inbox for a verification link. You can log in while you wait.";
  notice.hidden = false;
}

if (document.body.dataset.page === "account") {
  request("/api/auth/me").then(({ user }) => {
    document.getElementById("welcome").textContent = `Welcome, ${user.name}.`;
    document.getElementById("account-email").textContent = user.email;
    document.getElementById("verification-status").textContent = user.email_verified ? "Email verified" : "Email not verified — check your inbox.";
    const resend = document.getElementById("resend-account");
    resend.hidden = user.email_verified;
    resend.addEventListener("click", async () => {
      resend.disabled = true;
      try { await request("/api/auth/verification/resend", { email: user.email }); showNotice("Check your inbox for a new verification link."); }
      catch (error) { showError(error.message); }
      finally { resend.disabled = false; }
    });
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

function showNotice(message) {
  const notice = document.getElementById("notice");
  notice.textContent = message;
  notice.hidden = false;
}

const resendForm = document.getElementById("resend-form");
if (resendForm) resendForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const button = resendForm.querySelector("button"); button.disabled = true;
  try { const result = await request("/api/auth/verification/resend", Object.fromEntries(new FormData(resendForm))); showNotice(result.message); }
  catch (error) { showError(error.message); }
  finally { button.disabled = false; }
});

if (document.body.dataset.page === "verify") {
  const token = new URLSearchParams(window.location.hash.slice(1)).get("token");
  window.history.replaceState(null, "", "/verify-email");
  const button = document.getElementById("verify-email");
  if (!token) { button.disabled = true; showError("No verification token found. Request a new link from the login page."); }
  button.addEventListener("click", async () => {
    button.disabled = true;
    try { const result = await request("/api/auth/verify-email", { token }); showNotice(result.message); button.hidden = true; }
    catch (error) { showError(error.message); }
  });
}

const googleButton = document.getElementById("google-signin") || document.getElementById("google-link");
if (googleButton) {
  request("/api/auth/options").then((options) => { googleButton.hidden = !options.google_enabled; }).catch(() => {});
  googleButton.addEventListener("click", async () => {
    googleButton.disabled = true;
    try { const { url } = await request(googleButton.id === "google-link" ? "/api/auth/google/link" : "/api/auth/google/start", {}); window.location.assign(url); }
    catch (error) { showError(error.message); googleButton.disabled = false; }
  });
}

if (document.body.dataset.page === "login") {
	const errors = { google: "Google sign-in could not be completed. Please try again.", link_required: "This email already has an account. Log in with your password, verify your email, then connect Google from your account page.", reauth: "Log in again and verify your email before connecting Google.", verify_email: "Check your inbox and verify your email before signing in with Google." };
  const code = new URLSearchParams(window.location.search).get("error");
  if (errors[code]) showError(errors[code]);
}
if (document.body.dataset.page === "account" && new URLSearchParams(window.location.search).get("google") === "connected") showNotice("Google is connected. You can use it to sign in next time.");
