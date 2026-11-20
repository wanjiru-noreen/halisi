"use strict";

const list = document.getElementById("address-list");
const empty = document.getElementById("address-empty");
const form = document.getElementById("address-form");
const formError = document.getElementById("form-error");
const errorBox = document.getElementById("address-error");
const saveButton = document.getElementById("save-address");
const cancelButton = document.getElementById("cancel-edit");
const locateButton = document.getElementById("locate-me");
const geoStatus = document.getElementById("geo-status");
let addresses = [];
let editingID = null;

function showError(target, message) {
  target.textContent = message;
  target.hidden = false;
}

function showGeoStatus(message, isError = false) {
  if (!geoStatus) return;
  geoStatus.textContent = message;
  geoStatus.className = isError ? "mt-2 text-xs text-amber-800" : "mt-2 text-xs text-emerald-800 font-medium";
  geoStatus.hidden = false;
}

async function request(path, options = {}) {
  const response = await fetch(path, { credentials: "same-origin", cache: "no-store", ...options });
  const body = response.status === 204 ? {} : await response.json();
  if (response.status === 401) {
    window.location.replace("/login");
    throw new Error("Please log in to manage your addresses.");
  }
  if (!response.ok) throw new Error(body.error || "Address request could not be completed.");
  return body;
}

function addressCard(address) {
  const card = document.createElement("article");
  card.className = "rounded-2xl border border-stone-200 bg-white p-5 shadow-sm";
  const label = document.createElement("p");
  label.className = "text-xs font-bold uppercase tracking-[0.13em] text-emerald-700";
  label.textContent = address.label || "Address";
  const line = document.createElement("p");
  line.className = "mt-3 font-semibold text-stone-900";
  line.textContent = address.address_line;
  const city = document.createElement("p");
  city.className = "mt-1 text-sm text-stone-600";
  city.textContent = address.city;

  card.append(label, line, city);

  const hasCoords = address.latitude !== null && address.latitude !== undefined && address.longitude !== null && address.longitude !== undefined && String(address.latitude).trim() !== "" && String(address.longitude).trim() !== "";
  if (hasCoords) {
    const latNum = Number(address.latitude);
    const lngNum = Number(address.longitude);
    const coordsBadge = document.createElement("p");
    coordsBadge.className = "mt-2 inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-800";
    coordsBadge.textContent = `📍 ${latNum.toFixed(5)}, ${lngNum.toFixed(5)}`;
    card.append(coordsBadge);
  }

  const actions = document.createElement("div");
  actions.className = "mt-5 flex flex-wrap items-center gap-4";
  const edit = document.createElement("button");
  edit.type = "button";
  edit.className = "text-sm font-semibold text-emerald-800 hover:underline";
  edit.textContent = "Edit";
  edit.addEventListener("click", () => beginEdit(address));
  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "text-sm font-semibold text-red-700 hover:underline";
  remove.textContent = "Delete";
  remove.addEventListener("click", () => deleteAddress(address.id));

  const mapLink = document.createElement("a");
  mapLink.className = "text-sm font-semibold text-emerald-800 hover:underline inline-flex items-center gap-1";
  mapLink.target = "_blank";
  mapLink.rel = "noopener noreferrer";
  if (hasCoords) {
    mapLink.href = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address.latitude + "," + address.longitude)}`;
    mapLink.textContent = "View on Map ↗";
  } else {
    mapLink.href = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address.address_line + ", " + address.city)}`;
    mapLink.textContent = "Search on Map ↗";
  }

  actions.append(edit, remove, mapLink);
  card.append(actions);
  return card;
}

function render() {
  list.replaceChildren();
  empty.hidden = addresses.length > 0;
  addresses.forEach((address) => list.append(addressCard(address)));
}

function beginEdit(address) {
  editingID = address.id;
  form.elements.label.value = address.label || "";
  form.elements.address_line.value = address.address_line || "";
  form.elements.city.value = address.city || "";
  form.elements.latitude.value = address.latitude !== null && address.latitude !== undefined ? address.latitude : "";
  form.elements.longitude.value = address.longitude !== null && address.longitude !== undefined ? address.longitude : "";
  document.getElementById("form-eyebrow").textContent = "Update delivery details";
  document.getElementById("form-title").textContent = "Edit address";
  saveButton.textContent = "Save changes";
  cancelButton.classList.remove("hidden");
  if (geoStatus) geoStatus.hidden = true;
  form.scrollIntoView({ behavior: "smooth", block: "start" });
}

function resetForm() {
  editingID = null;
  form.reset();
  form.elements.latitude.value = "";
  form.elements.longitude.value = "";
  document.getElementById("form-eyebrow").textContent = "New delivery address";
  document.getElementById("form-title").textContent = "Add an address";
  saveButton.textContent = "Save address";
  cancelButton.classList.add("hidden");
  formError.hidden = true;
  if (geoStatus) geoStatus.hidden = true;
}

if (locateButton) {
  locateButton.addEventListener("click", () => {
    if (!navigator.geolocation) {
      showGeoStatus("Geolocation is not supported by your browser.", true);
      return;
    }
    showGeoStatus("Detecting current coordinates…");
    locateButton.disabled = true;
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        locateButton.disabled = false;
        const lat = pos.coords.latitude.toFixed(6);
        const lng = pos.coords.longitude.toFixed(6);
        form.elements.latitude.value = lat;
        form.elements.longitude.value = lng;
        showGeoStatus(`Location captured (${lat}, ${lng})`);
      },
      (err) => {
        locateButton.disabled = false;
        let msg = "Could not retrieve location.";
        if (err.code === 1) msg = "Location permission denied.";
        else if (err.code === 2) msg = "Location information is unavailable.";
        else if (err.code === 3) msg = "Location request timed out.";
        showGeoStatus(msg + " You can enter coordinates manually or leave them blank.", true);
      },
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 60000 }
    );
  });
}

async function loadAddresses() {
  try {
    const body = await request("/api/customer/addresses", { headers: { Accept: "application/json" } });
    addresses = body.addresses || [];
    render();
  } catch (error) {
    showError(errorBox, error.message);
  }
}

async function deleteAddress(id) {
  if (!window.confirm("Delete this saved address?")) return;
  errorBox.hidden = true;
  try {
    await request(`/api/customer/addresses/${encodeURIComponent(id)}`, { method: "DELETE", headers: { Accept: "application/json" } });
    addresses = addresses.filter((address) => Number(address.id) !== Number(id));
    render();
  } catch (error) {
    showError(errorBox, error.message);
  }
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  formError.hidden = true;
  saveButton.disabled = true;
  const payload = Object.fromEntries(new FormData(form));
  payload.label = String(payload.label || "").trim() || "Home";
  payload.address_line = String(payload.address_line || "").trim();
  payload.city = String(payload.city || "").trim();

  const latStr = String(payload.latitude || "").trim();
  const lngStr = String(payload.longitude || "").trim();
  if (latStr !== "" || lngStr !== "") {
    if (latStr === "" || lngStr === "") {
      showError(formError, "Provide both latitude and longitude, or leave both empty.");
      saveButton.disabled = false;
      return;
    }
    const lat = Number(latStr);
    const lng = Number(lngStr);
    if (Number.isNaN(lat) || lat < -90 || lat > 90) {
      showError(formError, "Latitude must be a valid number between -90 and 90.");
      saveButton.disabled = false;
      return;
    }
    if (Number.isNaN(lng) || lng < -180 || lng > 180) {
      showError(formError, "Longitude must be a valid number between -180 and 180.");
      saveButton.disabled = false;
      return;
    }
    payload.latitude = lat;
    payload.longitude = lng;
    payload.clear_coordinates = false;
  } else {
    delete payload.latitude;
    delete payload.longitude;
    if (editingID !== null) {
      payload.clear_coordinates = true;
    }
  }

  const method = editingID === null ? "POST" : "PATCH";
  const url = editingID === null ? "/api/customer/addresses" : `/api/customer/addresses/${encodeURIComponent(editingID)}`;
  try {
    const body = await request(url, { method, headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(payload) });
    if (editingID === null) addresses.push(body.address);
    else addresses = addresses.map((address) => Number(address.id) === Number(editingID) ? body.address : address);
    render();
    resetForm();
  } catch (error) {
    showError(formError, error.message);
  } finally {
    saveButton.disabled = false;
  }
});

cancelButton.addEventListener("click", resetForm);
loadAddresses();
