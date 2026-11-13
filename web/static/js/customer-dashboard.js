"use strict";

const errorBox = document.getElementById("error");
function showError(message) {
  errorBox.textContent = message;
  errorBox.hidden = false;
}
async function getJSON(path) {
  const response = await fetch(path, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
  const body = await response.json();
  if (response.status === 401) {
    window.location.replace("/login");
    throw new Error("Please log in to view your dashboard.");
  }
  if (!response.ok) throw new Error(body.error || "Could not load your account.");
  return body;
}

function displayOrder(order) {
  const row = document.createElement("a");
  row.href = `/customer/orders/${encodeURIComponent(order.id)}`;
  row.className = "flex flex-wrap items-center justify-between gap-3 py-4 hover:bg-stone-50";
  const left = document.createElement("div");
  const title = document.createElement("p");
  title.className = "font-semibold text-stone-900";
  title.textContent = `Order #${order.id}`;
  const date = document.createElement("p");
  date.className = "mt-1 text-sm text-stone-500";
  date.textContent = order.created_at ? new Date(order.created_at).toLocaleDateString() : "Recently placed";
  left.append(title, date);
  const right = document.createElement("div");
  right.className = "flex items-center gap-4";
  const status = document.createElement("span");
  status.className = "rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold capitalize text-emerald-800";
  status.textContent = String(order.status || "pending").replaceAll("_", " ");
  const total = document.createElement("span");
  total.className = "text-sm font-semibold text-stone-700";
  total.textContent = `${order.currency || "KES"} ${order.total ?? "—"}`;
  right.append(status, total);
  row.append(left, right);
  return row;
}

async function loadDashboard() {
  try {
    const [{ user }, { orders = [] }, { addresses = [] }] = await Promise.all([
      getJSON("/api/auth/me"),
      getJSON("/api/customer/orders"),
      getJSON("/api/customer/addresses"),
    ]);
    document.getElementById("welcome").textContent = `Welcome back, ${user.name}.`;
    document.getElementById("order-count").textContent = String(orders.length);
    document.getElementById("address-count").textContent = String(addresses.length);
    try {
      const cart = JSON.parse(localStorage.getItem("halisi-cart") || "[]");
      const cartCard = document.querySelector('a[href="/customer/cart"]');
      if (cartCard && Array.isArray(cart)) cartCard.querySelector("p.text-xl").textContent = `${cart.reduce((sum, item) => sum + Math.max(0, Number(item.quantity) || 0), 0)} items in your cart`;
    } catch { /* Ignore invalid local cart data and keep the default label. */ }
    const recent = document.getElementById("recent-orders");
    recent.replaceChildren();
    if (orders.length === 0) {
      const empty = document.createElement("p");
      empty.className = "py-5 text-sm text-stone-500";
      empty.textContent = "No orders yet. Find a local vendor to place your first order.";
      recent.append(empty);
      return;
    }
    orders.slice(0, 5).forEach((order) => recent.append(displayOrder(order)));
  } catch (error) {
    showError(error.message);
  }
}

document.getElementById("logout").addEventListener("click", async (event) => {
  event.currentTarget.disabled = true;
  try {
    const response = await fetch("/api/auth/logout", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: "{}" });
    if (!response.ok) throw new Error("Could not log out. Please try again.");
    window.location.replace("/login");
  } catch (error) {
    showError(error.message);
    event.currentTarget.disabled = false;
  }
});
loadDashboard();
