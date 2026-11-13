"use strict";

const list = document.getElementById("address-list");
const empty = document.getElementById("address-empty");
const form = document.getElementById("address-form");
const formError = document.getElementById("form-error");
const errorBox = document.getElementById("address-error");
const saveButton = document.getElementById("save-address");
const cancelButton = document.getElementById("cancel-edit");
let addresses = [];
let editingID = null;

function showError(target, message) {
  target.textContent = message;
  target.hidden = false;
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
  const actions = document.createElement("div");
  actions.className = "mt-5 flex gap-4";
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
  actions.append(edit, remove);
  card.append(label, line, city, actions);
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
  document.getElementById("form-eyebrow").textContent = "Update delivery details";
  document.getElementById("form-title").textContent = "Edit address";
  saveButton.textContent = "Save changes";
  cancelButton.classList.remove("hidden");
  form.scrollIntoView({ behavior: "smooth", block: "start" });
}

function resetForm() {
  editingID = null;
  form.reset();
  document.getElementById("form-eyebrow").textContent = "New delivery address";
  document.getElementById("form-title").textContent = "Add an address";
  saveButton.textContent = "Save address";
  cancelButton.classList.add("hidden");
  formError.hidden = true;
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
