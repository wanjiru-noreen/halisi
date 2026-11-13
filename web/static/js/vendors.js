"use strict";

const searchForm = document.getElementById("vendor-search");
const queryInput = document.getElementById("vendor-query");
const results = document.getElementById("vendor-results");
const errorBox = document.getElementById("vendor-error");

function vendorCard(vendor) {
  const card = document.createElement("article");
  card.className = "flex flex-col rounded-3xl border border-stone-200 bg-white p-6 shadow-sm";
  const badge = document.createElement("p");
  badge.className = "text-xs font-bold uppercase tracking-[0.14em] text-emerald-700";
  badge.textContent = "Approved HALISI vendor";
  const name = document.createElement("h3");
  name.className = "mt-3 text-xl font-semibold text-stone-900";
  name.textContent = vendor.name;
  const area = document.createElement("p");
  area.className = "mt-2 text-sm text-stone-600";
  area.textContent = vendor.service_area;
  const available = document.createElement("p");
  available.className = "mt-5 text-sm text-stone-600";
  available.textContent = vendor.product_count > 0 ? `${vendor.product_count} product${vendor.product_count === 1 ? "" : "s"} available` : "No products currently available";
  const link = document.createElement("a");
  link.className = "mt-6 inline-flex items-center gap-2 text-sm font-bold text-emerald-800 hover:underline";
  link.href = `/customer/products?vendor_id=${encodeURIComponent(vendor.id)}`;
  link.textContent = "Browse products →";
  card.append(badge, name, area, available, link);
  return card;
}

async function loadVendors(query = "") {
  errorBox.hidden = true;
  results.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "text-sm text-stone-500";
  loading.textContent = "Searching vendors…";
  results.append(loading);
  const params = new URLSearchParams();
  if (query.trim()) params.set("q", query.trim());
  try {
    const response = await fetch(`/api/storefront/vendors?${params}`, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "Could not load vendors.");
    results.replaceChildren();
    document.getElementById("result-count").textContent = `${body.vendors.length} ${body.vendors.length === 1 ? "vendor" : "vendors"}`;
    if (!body.vendors.length) {
      const empty = document.createElement("p");
      empty.className = "rounded-2xl border border-dashed border-stone-300 bg-white p-8 text-sm text-stone-600 sm:col-span-2 lg:col-span-3";
      empty.textContent = query.trim() ? "No approved vendors matched that search. Try another shop name or area." : "There are no approved vendors to show yet. Please check back soon.";
      results.append(empty);
      return;
    }
    body.vendors.forEach((vendor) => results.append(vendorCard(vendor)));
  } catch (error) {
    results.replaceChildren();
    errorBox.textContent = error.message;
    errorBox.hidden = false;
  }
}

searchForm.addEventListener("submit", (event) => {
  event.preventDefault();
  loadVendors(queryInput.value);
});
const initialQuery = new URLSearchParams(window.location.search).get("q") || "";
queryInput.value = initialQuery;
loadVendors(initialQuery);
