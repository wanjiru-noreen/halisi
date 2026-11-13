"use strict";

const productForm = document.getElementById("product-search");
const productQuery = document.getElementById("product-query");
const productResults = document.getElementById("product-results");
const productError = document.getElementById("product-error");
const selectedVendor = new URLSearchParams(window.location.search).get("vendor_id") || "";

function productCard(product) {
  const card = document.createElement("article");
  card.className = "flex flex-col rounded-3xl border border-stone-200 bg-white p-6 shadow-sm";
  const mark = document.createElement("div");
  mark.className = "flex h-14 w-14 items-center justify-center rounded-2xl bg-orange-50 text-2xl text-orange-700";
  mark.setAttribute("aria-hidden", "true");
  mark.textContent = "◉";
  const vendor = document.createElement("p");
  vendor.className = "mt-5 text-xs font-bold uppercase tracking-[0.14em] text-emerald-700";
  vendor.textContent = product.vendor_name;
  const name = document.createElement("h3");
  name.className = "mt-2 text-lg font-semibold text-stone-900";
  name.textContent = product.name;
  const price = document.createElement("p");
  price.className = "mt-4 text-2xl font-semibold text-stone-900";
  price.textContent = `KES ${product.price}`;
  const stock = document.createElement("p");
  stock.className = product.available_quantity > 0 ? "mt-2 text-sm text-emerald-700" : "mt-2 text-sm text-stone-500";
  stock.textContent = product.available_quantity > 0 ? `${product.available_quantity} available` : "Currently out of stock";
  const link = document.createElement("a");
  link.className = "mt-6 inline-flex items-center gap-2 text-sm font-bold text-emerald-800 hover:underline";
  link.href = `/customer/vendors?q=${encodeURIComponent(product.vendor_name)}`;
  link.textContent = "View vendor →";
  card.append(mark, vendor, name, price, stock, link);
  return card;
}

async function loadProducts(query = "") {
  productError.hidden = true;
  productResults.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "text-sm text-stone-500";
  loading.textContent = "Loading catalog…";
  productResults.append(loading);
  const params = new URLSearchParams();
  if (query.trim()) params.set("q", query.trim());
  if (/^[1-9][0-9]*$/.test(selectedVendor)) params.set("vendor_id", selectedVendor);
  try {
    const response = await fetch(`/api/storefront/products?${params}`, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "Could not load the product catalog.");
    productResults.replaceChildren();
    document.getElementById("product-count").textContent = `${body.products.length} ${body.products.length === 1 ? "product" : "products"}`;
    if (!body.products.length) {
      const empty = document.createElement("p");
      empty.className = "rounded-2xl border border-dashed border-stone-300 bg-white p-8 text-sm text-stone-600 sm:col-span-2 lg:col-span-3";
      empty.textContent = "No approved vendor products matched. Try another search or explore vendors.";
      productResults.append(empty);
      return;
    }
    body.products.forEach((product) => productResults.append(productCard(product)));
  } catch (error) {
    productResults.replaceChildren();
    productError.textContent = error.message;
    productError.hidden = false;
  }
}

productForm.addEventListener("submit", (event) => {
  event.preventDefault();
  loadProducts(productQuery.value);
});
loadProducts();
