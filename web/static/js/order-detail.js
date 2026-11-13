"use strict";

const error = document.getElementById("order-error");
const content = document.getElementById("order-content");
const orderID = window.location.pathname.split("/").filter(Boolean).at(-1);
let currentPayment = null;

function setText(id, value) { document.getElementById(id).textContent = value; }

function tracking(status, delivery) {
  const steps = document.getElementById("tracking-steps");
  const stages = ["pending", "confirmed", "preparing", "dispatched", "delivered"];
  const finalStatus = delivery?.status || status;
  const normalized = String(finalStatus || "pending").toLowerCase();
  const failed = ["failed", "cancelled"].includes(normalized);
  const current = stages.indexOf(normalized);
  const orderProgress = stages.indexOf(status);
  const index = current >= 0 ? current : orderProgress;
  stages.forEach((stage, position) => {
    const item = document.createElement("li");
    item.className = "flex items-center gap-3 rounded-xl border p-3 text-sm";
    const dot = document.createElement("span");
    dot.className = position <= index && !failed ? "h-3 w-3 rounded-full bg-emerald-600" : "h-3 w-3 rounded-full bg-stone-300";
    dot.setAttribute("aria-hidden", "true");
    const label = document.createElement("span");
    label.className = position <= index && !failed ? "font-semibold capitalize text-emerald-900" : "capitalize text-stone-500";
    label.textContent = stage;
    item.append(dot, label);
    steps.append(item);
  });
  const message = document.getElementById("tracking-message");
  if (failed) message.textContent = `This order is ${normalized.replaceAll("_", " ")}. Contact support or your vendor if you need help.`;
  else if (!delivery) message.textContent = "Your order has been recorded. Delivery updates will appear here when a rider is assigned.";
  else if (delivery.delivered_at) message.textContent = `Delivered on ${new Date(delivery.delivered_at).toLocaleString()}.`;
  else message.textContent = `Current delivery status: ${normalized.replaceAll("_", " ")}.`;
}

function addItem(item) {
  const row = document.createElement("div");
  row.className = "flex flex-wrap items-center justify-between gap-3 py-4";
  const name = document.createElement("p");
  name.className = "font-medium text-stone-900";
  name.textContent = item.product_name;
  const detail = document.createElement("p");
  detail.className = "mt-1 text-sm text-stone-500";
  detail.textContent = `${item.quantity} × KES ${item.unit_price}`;
  const line = document.createElement("p");
  line.className = "font-semibold text-stone-800";
  line.textContent = `KES ${(Number(item.quantity) * Number(item.unit_price)).toFixed(2)}`;
  const info = document.createElement("div");
  info.append(name, detail);
  row.append(info, line);
  document.getElementById("order-items").append(row);
}

async function loadOrder() {
  try {
    const response = await fetch(`/api/customer/orders/${encodeURIComponent(orderID)}`, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    const body = await response.json();
    if (response.status === 401) { window.location.replace("/login"); return; }
    if (!response.ok) throw new Error(body.error || "This order could not be loaded.");
    const order = body.order;
    document.getElementById("order-items").replaceChildren();
    document.getElementById("tracking-steps").replaceChildren();
    setText("order-title", `Order #${order.id}`);
    setText("order-date", new Date(order.created_at).toLocaleString());
    setText("order-status", String(order.status).replaceAll("_", " "));
    setText("vendor-name", order.vendor_name);
    setText("delivery-address", order.delivery_address);
    setText("order-total", `${order.currency} ${order.total}`);
    body.items.forEach(addItem);
    tracking(order.status, body.delivery);
    currentPayment = body.payment;
    const paymentPanel = document.getElementById("payment-panel");
    const availabilityResponse = await fetch("/api/mpesa/availability", { cache: "no-store", headers: { Accept: "application/json" } });
    const availability = availabilityResponse.ok ? await availabilityResponse.json() : { enabled: false };
    if (order.status === "pending" && availability.enabled) {
      paymentPanel.hidden = false;
      const paymentStatus = document.getElementById("payment-status");
      const processing = currentPayment && ["pending", "processing"].includes(currentPayment.status);
      paymentStatus.textContent = processing
        ? "Payment is processing. Complete the prompt on your phone if it arrived, then refresh this page to see the result."
        : currentPayment?.status === "failed"
          ? "The previous payment attempt failed. You can send a new payment prompt."
          : "Pay this order securely using M-Pesa. The amount is taken from the order total.";
      document.getElementById("payment-form").hidden = Boolean(processing);
    } else {
      paymentPanel.hidden = true;
    }
    content.hidden = false;
  } catch (err) {
    error.textContent = err.message;
    error.hidden = false;
  }
}

document.getElementById("mpesa-pay").addEventListener("click", async () => {
  const button = document.getElementById("mpesa-pay");
  const error = document.getElementById("payment-error");
  const phone = document.getElementById("mpesa-phone").value;
  error.hidden = true;
  button.disabled = true;
  button.textContent = "Sending prompt…";
  try {
    const response = await fetch(`/api/customer/orders/${encodeURIComponent(orderID)}/payments/mpesa`, {
      method: "POST",
      credentials: "same-origin",
      cache: "no-store",
      headers: { "Content-Type": "application/json", Accept: "application/json", "Idempotency-Key": crypto.randomUUID() },
      body: JSON.stringify({ phone_number: phone }),
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "M-Pesa could not start the payment.");
    await loadOrder();
  } catch (err) {
    error.textContent = err.message;
    error.hidden = false;
  } finally {
    button.disabled = false;
    button.textContent = "Send payment prompt";
  }
});

loadOrder();
