"use strict";

const ordersList = document.getElementById("orders-list");
const ordersEmpty = document.getElementById("orders-empty");
const ordersError = document.getElementById("orders-error");

function orderCard(order) {
  const link = document.createElement("a");
  link.href = `/customer/orders/${encodeURIComponent(order.id)}`;
  link.className = "block rounded-2xl border border-stone-200 bg-white p-5 shadow-sm transition hover:border-emerald-300 hover:shadow-md sm:p-6";
  const top = document.createElement("div");
  top.className = "flex flex-wrap items-center justify-between gap-4";
  const heading = document.createElement("div");
  const name = document.createElement("h2");
  name.className = "font-semibold text-stone-900";
  name.textContent = `Order #${order.id} · ${order.vendor_name}`;
  const date = document.createElement("p");
  date.className = "mt-1 text-sm text-stone-500";
  date.textContent = order.created_at ? new Date(order.created_at).toLocaleString() : "Date unavailable";
  heading.append(name, date);
  const badge = document.createElement("span");
  badge.className = "rounded-full bg-emerald-50 px-3 py-1 text-xs font-bold capitalize text-emerald-800";
  badge.textContent = String(order.status || "pending").replaceAll("_", " ");
  top.append(heading, badge);
  const bottom = document.createElement("div");
  bottom.className = "mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-stone-100 pt-4";
  const address = document.createElement("p");
  address.className = "text-sm text-stone-600";
  address.textContent = `Deliver to: ${order.delivery_address}`;
  const total = document.createElement("p");
  total.className = "text-sm font-semibold text-stone-900";
  total.textContent = `${order.currency} ${order.total}`;
  bottom.append(address, total);
  link.append(top, bottom);
  return link;
}

async function loadOrders() {
  try {
    const response = await fetch("/api/customer/orders", { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    const body = await response.json();
    if (response.status === 401) { window.location.replace("/login"); return; }
    if (!response.ok) throw new Error(body.error || "Could not load your orders.");
    ordersList.replaceChildren();
    const orders = body.orders || [];
    ordersEmpty.hidden = orders.length > 0;
    ordersList.hidden = orders.length === 0;
    orders.forEach((order) => ordersList.append(orderCard(order)));
  } catch (error) {
    ordersError.textContent = error.message;
    ordersError.hidden = false;
    ordersList.replaceChildren();
  }
}
loadOrders();
