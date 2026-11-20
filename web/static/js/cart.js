"use strict";

const cartKey = "halisi-cart";
const cartContainer = document.getElementById("cart-items");
const emptyMessage = document.getElementById("cart-empty");
const cartError = document.getElementById("cart-error");
const addressSelect = document.getElementById("delivery-address");
const checkoutButton = document.getElementById("checkout");
let cart = readCart();

function readCart() {
  try {
    const value = JSON.parse(localStorage.getItem(cartKey) || "[]");
    if (!Array.isArray(value)) return [];
    return value.filter((item) => item && Number.isSafeInteger(Number(item.product_id)) && Number(item.product_id) > 0 && Number.isSafeInteger(Number(item.quantity)) && Number(item.quantity) > 0);
  } catch {
    return [];
  }
}

function saveCart() {
  localStorage.setItem(cartKey, JSON.stringify(cart));
}

function money(value) {
  const amount = Number(value);
  return Number.isFinite(amount) && amount >= 0 ? `KES ${amount.toFixed(2)}` : "KES —";
}

function updateSummary() {
  const total = cart.reduce((sum, item) => sum + Math.max(0, Number(item.price) || 0) * Math.max(0, Number(item.quantity) || 0), 0);
  document.getElementById("cart-total").textContent = money(total);
  document.getElementById("cart-vendor").textContent = cart.length ? `From ${cart[0].vendor_name || "one vendor"}` : "";
  checkoutButton.disabled = cart.length === 0 || !addressSelect.value;
}

function renderCart() {
  cartContainer.replaceChildren();
  const isEmpty = cart.length === 0;
  emptyMessage.hidden = !isEmpty;
  cartContainer.hidden = isEmpty;
  if (isEmpty) {
    document.getElementById("cart-total").textContent = "KES 0.00";
    document.getElementById("cart-vendor").textContent = "";
    checkoutButton.disabled = true;
    return;
  }
  for (const item of cart) {
    const row = document.createElement("article");
    row.className = "flex flex-wrap items-center justify-between gap-4 py-5";
    const details = document.createElement("div");
    const title = document.createElement("h2");
    title.className = "font-semibold text-stone-900";
    title.textContent = item.name || `Product #${item.product_id}`;
    const vendor = document.createElement("p");
    vendor.className = "mt-1 text-sm text-stone-500";
    vendor.textContent = item.vendor_name || "HALISI vendor";
    const price = document.createElement("p");
    price.className = "mt-2 text-sm font-semibold text-stone-700";
    price.textContent = `${money(item.price)} each`;
    details.append(title, vendor, price);
    const actions = document.createElement("div");
    actions.className = "flex items-center gap-3";
    const quantity = document.createElement("input");
    quantity.type = "number";
    quantity.min = "1";
    quantity.max = "100";
    quantity.value = String(item.quantity);
    quantity.setAttribute("aria-label", `Quantity for ${item.name || "product"}`);
    quantity.className = "w-20 rounded-lg border border-stone-300 px-3 py-2 text-sm";
    quantity.addEventListener("change", () => {
      const value = Number.parseInt(quantity.value, 10);
      if (!Number.isInteger(value) || value < 1 || value > 100) {
        quantity.value = String(item.quantity);
        return;
      }
      item.quantity = value;
      saveCart();
      updateSummary();
    });
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "text-sm font-semibold text-red-700 hover:underline";
    remove.textContent = "Remove";
    remove.addEventListener("click", () => {
      cart = cart.filter((entry) => Number(entry.product_id) !== Number(item.product_id));
      saveCart();
      renderCart();
    });
    actions.append(quantity, remove);
    row.append(details, actions);
    cartContainer.append(row);
  }
  updateSummary();
}

async function loadAddresses() {
  addressSelect.replaceChildren();
  const prompt = document.createElement("option");
  prompt.value = "";
  prompt.textContent = "Choose a saved address";
  addressSelect.append(prompt);
  try {
    const response = await fetch("/api/customer/addresses", { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    const body = await response.json();
    if (response.status === 401) {
      document.getElementById("login-note").hidden = false;
      prompt.textContent = "Log in to load your saved addresses";
      return;
    }
    if (!response.ok) throw new Error(body.error || "Could not load saved addresses.");
    for (const address of body.addresses || []) {
      const option = document.createElement("option");
      option.value = String(address.id);
      option.textContent = `${address.label || "Address"}: ${address.address_line}, ${address.city}`;
      addressSelect.append(option);
    }
    if (!body.addresses || body.addresses.length === 0) {
      prompt.textContent = "Add an address to continue";
    }
  } catch (error) {
    prompt.textContent = "Addresses unavailable";
    cartError.textContent = error.message;
    cartError.hidden = false;
  }
  updateSummary();
}

addressSelect.addEventListener("change", updateSummary);
checkoutButton.addEventListener("click", async () => {
  if (!cart.length || !addressSelect.value) return;
  cartError.hidden = true;
  checkoutButton.disabled = true;
  checkoutButton.textContent = "Placing order…";
  try {
    const response = await fetch("/api/customer/checkout", {
      method: "POST",
      credentials: "same-origin",
      cache: "no-store",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ address_id: Number(addressSelect.value), items: cart.map((item) => ({ product_id: Number(item.product_id), quantity: Number(item.quantity) })) }),
    });
    const body = await response.json();
    if (response.status === 401) {
      document.getElementById("login-note").hidden = false;
      throw new Error("Log in to place your order.");
    }
    if (!response.ok) throw new Error(body.error || "Checkout could not be completed.");
    localStorage.setItem(cartKey, "[]");
    window.location.assign(`/customer/orders/${encodeURIComponent(body.order.id)}`);
  } catch (error) {
    cartError.textContent = error.message;
    cartError.hidden = false;
    checkoutButton.textContent = "Place order";
    updateSummary();
  }
});

renderCart();
loadAddresses();
