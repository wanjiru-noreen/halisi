"use strict";

const errorBox = document.getElementById("error");
function showError(message) { errorBox.textContent = message; errorBox.hidden = false; }
async function api(path, logout = false) {
  const response = await fetch(path, { method: logout ? "POST" : "GET", credentials: "same-origin", cache: "no-store", headers: logout ? { "Content-Type": "application/json" } : {}, body: logout ? "{}" : undefined });
  if (response.status === 401) { window.location.replace("/login"); throw new Error("Please log in."); }
  const value = response.status === 204 ? {} : await response.json();
  if (!response.ok) throw new Error(value.error || "Couldn't load this resource.");
  return value;
}
const sections = {
  customer: [["Your orders", "/api/customer/orders", "orders"], ["Saved addresses", "/api/customer/addresses", "addresses"], ["Your payments", "/api/customer/payments", "payments"]],
  vendor: [["Your shop", "/api/vendor/shop", "shop"], ["Your products", "/api/vendor/products", "products"], ["Your shop's orders", "/api/vendor/orders", "orders"]],
  rider: [["Your assigned deliveries", "/api/rider/deliveries", "deliveries"]],
  admin: [["Users", "/api/admin/users", "users"], ["Vendor management", "/api/admin/vendors", "vendors"]],
};
const roleNames = { customer: "Customer", vendor: "Vendor", rider: "Rider", admin: "Administrator" };

async function loadSection([title, path, key]) {
  const card = document.createElement("section"); card.className = "rounded-3xl border border-stone-200 bg-white p-7 shadow-sm";
  const heading = document.createElement("h2"); heading.className = "text-xl font-semibold"; heading.textContent = title;
  const list = document.createElement("ul"); list.className = "mt-5 space-y-3 text-sm text-stone-600";
  card.append(heading, list); document.getElementById("sections").append(card);
  try {
    const result = await api(path);
    const rows = Array.isArray(result[key]) ? result[key] : [result[key]];
    if (!rows.length) { const empty = document.createElement("li"); empty.textContent = "Nothing here yet."; list.append(empty); }
    for (const row of rows) {
      const item = document.createElement("li"); item.className = "border-t border-stone-100 py-3";
      const detail = [row.name || row.label || `#${row.id}`, row.status, row.role, row.email, row.address_line, row.city, row.total !== undefined ? `${row.currency} ${row.total}` : "", row.price !== undefined ? `Price ${row.price}` : "", row.delivery_address].filter(Boolean);
      item.textContent = detail.join(" · "); list.append(item);
    }
  } catch (error) { const item = document.createElement("li"); item.textContent = error.message; list.append(item); }
}
api("/api/auth/me").then(async ({ user }) => {
  if (!sections[user.role]) throw new Error("Your role has no permitted dashboard.");
  document.getElementById("role-label").textContent = `${roleNames[user.role]} dashboard`;
  document.getElementById("welcome").textContent = `Welcome, ${user.name}.`;
  await Promise.all(sections[user.role].map(loadSection));
}).catch((error) => showError(error.message));
document.getElementById("logout").addEventListener("click", async (event) => {
  event.currentTarget.disabled = true;
  try { await api("/api/auth/logout", true); window.location.replace("/login"); }
  catch (error) { showError(error.message); document.getElementById("logout").disabled = false; }
});
