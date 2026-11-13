"use strict";
const errorBox = document.getElementById("error");
const fail = (error) => { errorBox.textContent = error.message || "Please try again."; errorBox.hidden = false; };
async function api(path, method = "GET", body) {
  const multipart = body instanceof FormData;
  const response = await fetch(path, { method, credentials: "same-origin", cache: "no-store", headers: body && !multipart ? { "Content-Type": "application/json" } : {}, body: body ? (multipart ? body : JSON.stringify(body)) : undefined });
  if (response.status === 401) { window.location.replace("/login"); throw new Error("Please log in."); }
  const result = await response.json(); if (!response.ok) throw new Error(result.error || "Request failed."); return result;
}
function documents(rows) {
  const container = document.getElementById("documents"); container.replaceChildren();
  for (const row of rows) { const link = document.createElement("a"); link.className = "block text-sm text-emerald-800 underline"; link.href = `/api/vendor/documents/${row.id}`; link.textContent = `${row.document_type}: ${row.original_name} (${row.status})`; container.append(link); }
}
const page = document.body.dataset.vendorPage;
if (page === "register") (async () => {
  const [{ user }, { application }] = await Promise.all([api("/api/auth/me"), api("/api/vendor/application")]);
  const form = document.getElementById("business-form");
  if (!user.email_verified) { for (const field of form.elements) field.disabled = true; throw new Error("Verify your email from your account page before registering your business."); }
  for (const name of ["name", "contact_email", "contact_phone", "address", "city"]) form.elements[name].value = application?.[name] || (name === "contact_email" ? user.email : name === "contact_phone" ? user.phone : "");
  for (const day of ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]) {
    const hours = application?.operating_hours?.[day] || { open: "08:00", close: "18:00", closed: false };
    const row = document.createElement("div"); row.className = "flex flex-wrap items-center gap-3";
    const title = document.createElement("span"); title.className = "w-12 text-sm font-semibold"; title.textContent = day.toUpperCase(); row.append(title);
    for (const key of ["open", "close"]) { const input = document.createElement("input"); input.type = "time"; input.name = `${day}-${key}`; input.value = hours[key] || ""; input.className = "rounded-lg border border-stone-300 p-2"; input.setAttribute("aria-label", `${day} ${key}`); row.append(input); }
    const label = document.createElement("label"); label.className = "text-sm"; const closed = document.createElement("input"); closed.type = "checkbox"; closed.name = `${day}-closed`; closed.checked = hours.closed; label.append(closed, " Closed"); row.append(label); document.getElementById("hours").append(row);
  }
  form.addEventListener("submit", async (event) => { event.preventDefault(); const data = {}; for (const name of ["name", "contact_email", "contact_phone", "address", "city"]) data[name] = form.elements[name].value;
    data.operating_hours = {}; for (const day of ["mon", "tue", "wed", "thu", "fri", "sat", "sun"]) { const closed = form.elements[`${day}-closed`].checked; data.operating_hours[day] = { closed, open: closed ? "" : form.elements[`${day}-open`].value, close: closed ? "" : form.elements[`${day}-close`].value }; }
    const button = form.querySelector("button"); button.disabled = true; try { await api("/api/vendor/application", "PUT", data); window.location.assign("/vendor/verification"); } catch (error) { fail(error); button.disabled = false; }
  });
})().catch(fail);
if (page === "status") {
  api("/api/vendor/application").then(({ application, documents: rows }) => { if (!application) { window.location.replace("/vendor/register"); return; } document.getElementById("status").textContent = `${application.name}: ${application.status}${application.is_active ? " · active" : " · not selling"}${!application.submitted_at ? " · not submitted" : ""}`; document.getElementById("reason").textContent = application.review_reason; documents(rows); if (application.status === "suspended") for (const button of document.querySelectorAll("button")) button.disabled = true; }).catch(fail);
  for (const form of document.querySelectorAll(".upload-form")) form.addEventListener("submit", async (event) => { event.preventDefault(); const file = form.elements.document.files[0]; if (!file || file.size > 5 * 1024 * 1024) { fail(new Error("Choose a PDF, JPEG or PNG up to 5 MiB.")); return; } const button = form.querySelector("button"); button.disabled = true; try { await api(`/api/vendor/documents/${form.dataset.type}`, "POST", new FormData(form)); window.location.reload(); } catch (error) { fail(error); button.disabled = false; } });
  document.getElementById("submit-application").addEventListener("click", async (event) => { const button = event.currentTarget; button.disabled = true; try { await api("/api/vendor/application/submit", "POST", {}); window.location.reload(); } catch (error) { fail(error); button.disabled = false; } });
}
if (page === "review") {
  let current;
  api("/api/admin/vendors").then(({ vendors }) => { for (const vendor of vendors) { const button = document.createElement("button"); button.className = "rounded-xl border border-stone-300 bg-white px-4 py-2 text-sm"; button.textContent = `${vendor.name} · ${vendor.status}`; button.addEventListener("click", async () => { try { const result = await api(`/api/admin/vendor-applications/${vendor.id}`); current = vendor.id; document.getElementById("business-name").textContent = result.application.name; document.getElementById("business-details").textContent = JSON.stringify(result.application, null, 2); documents(result.documents); document.getElementById("review-panel").hidden = false; } catch (error) { fail(error); } }); document.getElementById("applications").append(button); } }).catch(fail);
  document.getElementById("review-form").addEventListener("submit", async (event) => { event.preventDefault(); if (!current) return; const button = event.currentTarget.querySelector("button"); button.disabled = true; try { await api(`/api/admin/vendor-applications/${current}/review`, "POST", Object.fromEntries(new FormData(event.currentTarget))); window.location.reload(); } catch (error) { fail(error); button.disabled = false; } });
}
