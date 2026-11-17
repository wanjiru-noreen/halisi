"use strict";

// DOM references
const errorBanner = document.getElementById("error-banner");
const successBanner = document.getElementById("success-banner");
const welcomeHeading = document.getElementById("welcome");
const riderStatusLine = document.getElementById("rider-status-line");
const activeList = document.getElementById("active-list");
const activeBadge = document.getElementById("active-badge");
const deliveriesList = document.getElementById("deliveries-list");
const searchInput = document.getElementById("search-input");
const filterTabs = document.querySelectorAll(".filter-tab");
const refreshBtn = document.getElementById("refresh-button");
const logoutBtn = document.getElementById("logout");

// Metric cards
const statAssigned = document.getElementById("stat-assigned");
const statPickedUp = document.getElementById("stat-picked-up");
const statDelivered = document.getElementById("stat-delivered");
const statTotal = document.getElementById("stat-total");

// Modal elements
const detailsModal = document.getElementById("details-modal");
const modalClose = document.getElementById("modal-close");
const modalTitle = document.getElementById("modal-title");
const modalPickup = document.getElementById("modal-pickup");
const modalPickupMap = document.getElementById("modal-pickup-map");
const modalDropoff = document.getElementById("modal-dropoff");
const modalDropoffMap = document.getElementById("modal-dropoff-map");
const modalPhone = document.getElementById("modal-phone");
const modalPhoneLink = document.getElementById("modal-phone-link");
const modalStatus = document.getElementById("modal-status");
const modalDate = document.getElementById("modal-date");
const modalActions = document.getElementById("modal-actions");
const stepAssigned = document.getElementById("step-assigned");
const stepPickedUp = document.getElementById("step-picked-up");
const stepDelivered = document.getElementById("step-delivered");

// Issue confirmation modal
const issueModal = document.getElementById("issue-modal");
const issueCancel = document.getElementById("issue-cancel");
const issueConfirm = document.getElementById("issue-confirm");

// State
let allDeliveries = [];
let currentFilter = "all";
let currentSearch = "";
let pendingIssueDeliveryId = null;

function showError(msg) {
  if (!errorBanner) return;
  errorBanner.textContent = msg;
  errorBanner.hidden = false;
  if (successBanner) successBanner.hidden = true;
}

function showSuccess(msg) {
  if (!successBanner) return;
  successBanner.textContent = msg;
  successBanner.hidden = false;
  if (errorBanner) errorBanner.hidden = true;
  setTimeout(() => {
    if (successBanner && successBanner.textContent === msg) {
      successBanner.hidden = true;
    }
  }, 5000);
}

function clearAlerts() {
  if (errorBanner) errorBanner.hidden = true;
  if (successBanner) successBanner.hidden = true;
}

async function api(path, options = {}) {
  const method = options.method || "GET";
  const headers = { Accept: "application/json", ...(options.headers || {}) };
  if (options.body && typeof options.body === "object" && !(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
    options.body = JSON.stringify(options.body);
  }
  const res = await fetch(path, {
    method,
    credentials: "same-origin",
    cache: "no-store",
    headers,
    body: options.body,
  });

  if (res.status === 401) {
    window.location.replace("/login");
    throw new Error("Session expired. Please log in.");
  }

  const data = res.status === 204 ? {} : await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `Request failed with status ${res.status}`);
  }
  return data;
}

function createBadge(status) {
  const badge = document.createElement("span");
  badge.className = "rounded-full px-3 py-1 text-xs font-bold capitalize ";
  switch (status) {
    case "assigned":
      badge.className += "bg-orange-50 text-orange-800 border border-orange-200";
      badge.textContent = "Ready for Pickup";
      break;
    case "picked_up":
      badge.className += "bg-emerald-100 text-emerald-900 border border-emerald-300";
      badge.textContent = "In Transit";
      break;
    case "delivered":
      badge.className += "bg-emerald-50 text-emerald-800";
      badge.textContent = "Delivered";
      break;
    case "failed":
      badge.className += "bg-red-50 text-red-800";
      badge.textContent = "Failed";
      break;
    case "cancelled":
      badge.className += "bg-stone-200 text-stone-700";
      badge.textContent = "Cancelled";
      break;
    default:
      badge.className += "bg-stone-100 text-stone-700";
      badge.textContent = String(status || "Pending").replaceAll("_", " ");
  }
  return badge;
}

function mapUrl(address) {
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address || "")}`;
}

function renderActiveCard(delivery) {
  const card = document.createElement("article");
  card.className = "rounded-2xl border-2 border-emerald-600/30 bg-emerald-50/20 p-5 shadow-sm transition hover:border-emerald-600/60";

  // Header row
  const top = document.createElement("div");
  top.className = "flex flex-wrap items-center justify-between gap-3 border-b border-stone-200/60 pb-3";

  const titleGroup = document.createElement("div");
  const title = document.createElement("h3");
  title.className = "font-bold text-stone-900";
  title.textContent = `Delivery #${delivery.id} · Order #${delivery.order_id}`;
  const date = document.createElement("p");
  date.className = "mt-0.5 text-xs text-stone-500";
  date.textContent = delivery.created_at ? `Assigned on ${new Date(delivery.created_at).toLocaleString()}` : "";
  titleGroup.append(title, date);

  top.append(titleGroup, createBadge(delivery.status));
  card.append(top);

  // Address info
  const details = document.createElement("div");
  details.className = "mt-4 grid gap-3 sm:grid-cols-2 text-sm";

  const pickupCol = document.createElement("div");
  pickupCol.className = "rounded-xl bg-white p-3 border border-stone-200";
  const pickupLabel = document.createElement("p");
  pickupLabel.className = "text-xs font-bold uppercase tracking-wider text-stone-500";
  pickupLabel.textContent = "Pickup Location (Vendor)";
  const pickupText = document.createElement("p");
  pickupText.className = "mt-1 font-medium text-stone-900";
  pickupText.textContent = delivery.pickup_address || "Shop address";
  const pickupLink = document.createElement("a");
  pickupLink.href = mapUrl(delivery.pickup_address);
  pickupLink.target = "_blank";
  pickupLink.rel = "noopener noreferrer";
  pickupLink.className = "mt-2 inline-block text-xs font-semibold text-emerald-800 hover:underline";
  pickupLink.textContent = "Map Directions ↗";
  pickupCol.append(pickupLabel, pickupText, pickupLink);

  const dropoffCol = document.createElement("div");
  dropoffCol.className = "rounded-xl bg-white p-3 border border-stone-200";
  const dropoffLabel = document.createElement("p");
  dropoffLabel.className = "text-xs font-bold uppercase tracking-wider text-stone-500";
  dropoffLabel.textContent = "Dropoff Address (Customer)";
  const dropoffText = document.createElement("p");
  dropoffText.className = "mt-1 font-medium text-stone-900";
  dropoffText.textContent = delivery.delivery_address || "Customer address";
  const dropoffLink = document.createElement("a");
  dropoffLink.href = mapUrl(delivery.delivery_address);
  dropoffLink.target = "_blank";
  dropoffLink.rel = "noopener noreferrer";
  dropoffLink.className = "mt-2 inline-block text-xs font-semibold text-emerald-800 hover:underline";
  dropoffLink.textContent = "Map Directions ↗";
  dropoffCol.append(dropoffLabel, dropoffText, dropoffLink);

  details.append(pickupCol, dropoffCol);
  card.append(details);

  // Customer contact phone
  if (delivery.phone) {
    const phoneRow = document.createElement("div");
    phoneRow.className = "mt-3 flex items-center justify-between rounded-xl bg-stone-100 px-3 py-2 text-sm";
    const phoneLabel = document.createElement("span");
    phoneLabel.className = "text-xs font-semibold text-stone-600";
    phoneLabel.textContent = "Customer Contact:";
    const phoneLink = document.createElement("a");
    phoneLink.href = `tel:${encodeURIComponent(delivery.phone)}`;
    phoneLink.className = "font-bold text-emerald-800 hover:underline";
    phoneLink.textContent = `📞 ${delivery.phone}`;
    phoneRow.append(phoneLabel, phoneLink);
    card.append(phoneRow);
  }

  // Action Buttons
  const actions = document.createElement("div");
  actions.className = "mt-4 flex flex-wrap items-center justify-end gap-2 border-t border-stone-200/60 pt-3";

  const detailsBtn = document.createElement("button");
  detailsBtn.className = "rounded-full border border-stone-300 bg-white px-4 py-2 text-xs font-semibold text-stone-700 hover:bg-stone-50";
  detailsBtn.textContent = "View Details";
  detailsBtn.addEventListener("click", () => openDetails(delivery));
  actions.append(detailsBtn);

  if (delivery.status === "assigned") {
    const pickupActionBtn = document.createElement("button");
    pickupActionBtn.className = "rounded-full bg-emerald-800 px-4 py-2 text-xs font-bold text-white hover:bg-emerald-900";
    pickupActionBtn.textContent = "Confirm Gas Pickup";
    pickupActionBtn.addEventListener("click", () => handleStatusChange(delivery.id, "picked_up", pickupActionBtn));

    const issueActionBtn = document.createElement("button");
    issueActionBtn.className = "rounded-full border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 hover:bg-red-100";
    issueActionBtn.textContent = "Report Issue";
    issueActionBtn.addEventListener("click", () => promptIssue(delivery.id));

    actions.append(issueActionBtn, pickupActionBtn);
  } else if (delivery.status === "picked_up") {
    const deliveredActionBtn = document.createElement("button");
    deliveredActionBtn.className = "rounded-full bg-emerald-800 px-4 py-2 text-xs font-bold text-white hover:bg-emerald-900";
    deliveredActionBtn.textContent = "Confirm Gas Delivered";
    deliveredActionBtn.addEventListener("click", () => handleStatusChange(delivery.id, "delivered", deliveredActionBtn));

    const issueActionBtn = document.createElement("button");
    issueActionBtn.className = "rounded-full border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 hover:bg-red-100";
    issueActionBtn.textContent = "Report Issue";
    issueActionBtn.addEventListener("click", () => promptIssue(delivery.id));

    actions.append(issueActionBtn, deliveredActionBtn);
  }

  card.append(actions);
  return card;
}

function renderDeliveryRow(delivery) {
  const row = document.createElement("div");
  row.className = "rounded-2xl border border-stone-200 bg-white p-4 shadow-sm transition hover:border-emerald-300 sm:p-5";

  const top = document.createElement("div");
  top.className = "flex flex-wrap items-center justify-between gap-3";

  const left = document.createElement("div");
  const title = document.createElement("h3");
  title.className = "font-semibold text-stone-900";
  title.textContent = `Delivery #${delivery.id} · Order #${delivery.order_id}`;

  const date = document.createElement("p");
  date.className = "mt-0.5 text-xs text-stone-500";
  date.textContent = delivery.delivered_at
    ? `Delivered on ${new Date(delivery.delivered_at).toLocaleString()}`
    : delivery.created_at
      ? `Assigned on ${new Date(delivery.created_at).toLocaleString()}`
      : "";
  left.append(title, date);

  top.append(left, createBadge(delivery.status));
  row.append(top);

  const middle = document.createElement("div");
  middle.className = "mt-3 grid gap-2 sm:grid-cols-2 text-xs text-stone-600";
  const pickupP = document.createElement("p");
  pickupP.textContent = `Pickup: ${delivery.pickup_address || "Shop"}`;
  const dropoffP = document.createElement("p");
  dropoffP.textContent = `Deliver to: ${delivery.delivery_address || "Customer address"}`;
  middle.append(pickupP, dropoffP);
  row.append(middle);

  const bottom = document.createElement("div");
  bottom.className = "mt-4 flex flex-wrap items-center justify-between gap-2 border-t border-stone-100 pt-3";

  const phoneSpan = document.createElement("span");
  phoneSpan.className = "text-xs text-stone-500";
  phoneSpan.textContent = delivery.phone ? `Customer: ${delivery.phone}` : "";

  const btnGroup = document.createElement("div");
  btnGroup.className = "flex items-center gap-2";

  const detailsBtn = document.createElement("button");
  detailsBtn.className = "rounded-full border border-stone-300 px-3 py-1.5 text-xs font-semibold hover:bg-stone-50";
  detailsBtn.textContent = "Details";
  detailsBtn.addEventListener("click", () => openDetails(delivery));
  btnGroup.append(detailsBtn);

  if (delivery.status === "assigned") {
    const pickupBtn = document.createElement("button");
    pickupBtn.className = "rounded-full bg-emerald-800 px-3 py-1.5 text-xs font-bold text-white hover:bg-emerald-900";
    pickupBtn.textContent = "Pickup";
    pickupBtn.addEventListener("click", () => handleStatusChange(delivery.id, "picked_up", pickupBtn));
    btnGroup.append(pickupBtn);
  } else if (delivery.status === "picked_up") {
    const deliverBtn = document.createElement("button");
    deliverBtn.className = "rounded-full bg-emerald-800 px-3 py-1.5 text-xs font-bold text-white hover:bg-emerald-900";
    deliverBtn.textContent = "Delivered";
    deliverBtn.addEventListener("click", () => handleStatusChange(delivery.id, "delivered", deliverBtn));
    btnGroup.append(deliverBtn);
  }

  bottom.append(phoneSpan, btnGroup);
  row.append(bottom);
  return row;
}

function updateStats(deliveries) {
  const assigned = deliveries.filter((d) => d.status === "assigned").length;
  const pickedUp = deliveries.filter((d) => d.status === "picked_up").length;
  const delivered = deliveries.filter((d) => d.status === "delivered").length;
  const total = deliveries.length;

  if (statAssigned) statAssigned.textContent = String(assigned);
  if (statPickedUp) statPickedUp.textContent = String(pickedUp);
  if (statDelivered) statDelivered.textContent = String(delivered);
  if (statTotal) statTotal.textContent = String(total);

  const activeCount = assigned + pickedUp;
  if (activeBadge) activeBadge.textContent = `${activeCount} active`;
}

function renderUI() {
  updateStats(allDeliveries);

  // 1. Active Deliveries
  const activeDeliveries = allDeliveries.filter((d) => d.status === "assigned" || d.status === "picked_up");
  if (activeList) {
    activeList.replaceChildren();
    if (activeDeliveries.length === 0) {
      const empty = document.createElement("div");
      empty.className = "rounded-2xl border border-dashed border-stone-200 p-8 text-center";
      const p = document.createElement("p");
      p.className = "font-medium text-stone-700";
      p.textContent = "No active deliveries right now.";
      const sub = document.createElement("p");
      sub.className = "mt-1 text-xs text-stone-500";
      sub.textContent = "When a vendor confirms an order, your new delivery assignments will appear here.";
      empty.append(p, sub);
      activeList.append(empty);
    } else {
      activeDeliveries.forEach((d) => activeList.append(renderActiveCard(d)));
    }
  }

  // 2. Filtered list
  if (deliveriesList) {
    deliveriesList.replaceChildren();
    let filtered = allDeliveries;

    if (currentFilter === "active") {
      filtered = filtered.filter((d) => d.status === "assigned" || d.status === "picked_up");
    } else if (currentFilter === "delivered") {
      filtered = filtered.filter((d) => d.status === "delivered");
    } else if (currentFilter === "failed") {
      filtered = filtered.filter((d) => d.status === "failed" || d.status === "cancelled");
    }

    if (currentSearch.trim()) {
      const q = currentSearch.trim().toLowerCase();
      filtered = filtered.filter((d) => {
        return (
          String(d.id).includes(q) ||
          String(d.order_id).includes(q) ||
          (d.pickup_address && d.pickup_address.toLowerCase().includes(q)) ||
          (d.delivery_address && d.delivery_address.toLowerCase().includes(q)) ||
          (d.phone && d.phone.toLowerCase().includes(q))
        );
      });
    }

    if (filtered.length === 0) {
      const empty = document.createElement("div");
      empty.className = "py-8 text-center text-sm text-stone-500";
      empty.textContent = allDeliveries.length === 0
        ? "No deliveries assigned to your account yet."
        : "No deliveries match your selected filters.";
      deliveriesList.append(empty);
    } else {
      filtered.forEach((d) => deliveriesList.append(renderDeliveryRow(d)));
    }
  }
}

async function handleStatusChange(deliveryId, newStatus, btn) {
  clearAlerts();
  const originalText = btn ? btn.textContent : "";
  if (btn) {
    btn.disabled = true;
    btn.textContent = "Updating…";
  }

  try {
    const res = await api(`/api/rider/deliveries/${deliveryId}`, {
      method: "PATCH",
      body: { status: newStatus },
    });

    const updated = res.delivery;
    const index = allDeliveries.findIndex((d) => d.id === deliveryId);
    if (index >= 0) {
      allDeliveries[index].status = updated.status;
      allDeliveries[index].delivered_at = updated.delivered_at;
      allDeliveries[index].updated_at = updated.updated_at;
    }

    showSuccess(`Delivery #${deliveryId} successfully updated to ${newStatus.replaceAll("_", " ")}.`);
    renderUI();

    // If details modal is open for this delivery, update it too
    if (detailsModal && !detailsModal.hidden && modalTitle.textContent.includes(`#${deliveryId}`)) {
      openDetails(allDeliveries[index]);
    }
  } catch (err) {
    showError(err.message || "Failed to update delivery status.");
    if (btn) {
      btn.disabled = false;
      btn.textContent = originalText;
    }
  }
}

function promptIssue(deliveryId) {
  pendingIssueDeliveryId = deliveryId;
  if (issueModal) issueModal.hidden = false;
}

if (issueCancel) {
  issueCancel.addEventListener("click", () => {
    pendingIssueDeliveryId = null;
    if (issueModal) issueModal.hidden = true;
  });
}

if (issueConfirm) {
  issueConfirm.addEventListener("click", async () => {
    if (!pendingIssueDeliveryId) return;
    const targetId = pendingIssueDeliveryId;
    pendingIssueDeliveryId = null;
    if (issueModal) issueModal.hidden = true;
    await handleStatusChange(targetId, "failed", null);
  });
}

function openDetails(delivery) {
  if (!detailsModal) return;
  modalTitle.textContent = `Delivery #${delivery.id} (Order #${delivery.order_id})`;
  modalPickup.textContent = delivery.pickup_address || "Vendor location";
  modalPickupMap.href = mapUrl(delivery.pickup_address);
  modalDropoff.textContent = delivery.delivery_address || "Customer address";
  modalDropoffMap.href = mapUrl(delivery.delivery_address);

  modalPhone.textContent = delivery.phone || "Not specified";
  if (delivery.phone) {
    modalPhoneLink.href = `tel:${encodeURIComponent(delivery.phone)}`;
    modalPhoneLink.hidden = false;
  } else {
    modalPhoneLink.hidden = true;
  }

  modalStatus.textContent = String(delivery.status || "").replaceAll("_", " ");
  modalDate.textContent = delivery.delivered_at
    ? `Completed at: ${new Date(delivery.delivered_at).toLocaleString()}`
    : delivery.created_at
      ? `Assigned: ${new Date(delivery.created_at).toLocaleString()}`
      : "";

  // Stepper highlight
  const norm = String(delivery.status || "pending").toLowerCase();
  const isFailed = norm === "failed" || norm === "cancelled";

  stepAssigned.className = "rounded-xl border p-2 ";
  stepPickedUp.className = "rounded-xl border p-2 ";
  stepDelivered.className = "rounded-xl border p-2 ";

  if (isFailed) {
    stepAssigned.className += "bg-red-50 text-red-800 border-red-300";
    stepPickedUp.className += "bg-stone-100 text-stone-400";
    stepDelivered.className += "bg-stone-100 text-stone-400";
  } else {
    stepAssigned.className += norm !== "pending" ? "bg-emerald-100 text-emerald-900 border-emerald-400" : "bg-stone-100 text-stone-500";
    stepPickedUp.className += norm === "picked_up" || norm === "delivered" ? "bg-emerald-100 text-emerald-900 border-emerald-400" : "bg-stone-100 text-stone-500";
    stepDelivered.className += norm === "delivered" ? "bg-emerald-600 text-white font-bold" : "bg-stone-100 text-stone-500";
  }

  // Modal action buttons
  modalActions.replaceChildren();
  const closeBtn = document.createElement("button");
  closeBtn.className = "rounded-full border border-stone-300 px-4 py-2 text-sm font-semibold hover:bg-stone-100";
  closeBtn.textContent = "Close";
  closeBtn.addEventListener("click", closeDetails);
  modalActions.append(closeBtn);

  if (delivery.status === "assigned") {
    const issueBtn = document.createElement("button");
    issueBtn.className = "rounded-full border border-red-200 bg-red-50 px-4 py-2 text-sm font-semibold text-red-700 hover:bg-red-100";
    issueBtn.textContent = "Report Issue";
    issueBtn.addEventListener("click", () => promptIssue(delivery.id));

    const pickupBtn = document.createElement("button");
    pickupBtn.className = "rounded-full bg-emerald-800 px-5 py-2 text-sm font-bold text-white hover:bg-emerald-900";
    pickupBtn.textContent = "Confirm Gas Pickup";
    pickupBtn.addEventListener("click", () => handleStatusChange(delivery.id, "picked_up", pickupBtn));

    modalActions.append(issueBtn, pickupBtn);
  } else if (delivery.status === "picked_up") {
    const issueBtn = document.createElement("button");
    issueBtn.className = "rounded-full border border-red-200 bg-red-50 px-4 py-2 text-sm font-semibold text-red-700 hover:bg-red-100";
    issueBtn.textContent = "Report Issue";
    issueBtn.addEventListener("click", () => promptIssue(delivery.id));

    const deliverBtn = document.createElement("button");
    deliverBtn.className = "rounded-full bg-emerald-800 px-5 py-2 text-sm font-bold text-white hover:bg-emerald-900";
    deliverBtn.textContent = "Confirm Gas Delivered";
    deliverBtn.addEventListener("click", () => handleStatusChange(delivery.id, "delivered", deliverBtn));

    modalActions.append(issueBtn, deliverBtn);
  }

  detailsModal.hidden = false;
}

function closeDetails() {
  if (detailsModal) detailsModal.hidden = true;
}

if (modalClose) modalClose.addEventListener("click", closeDetails);

// Close modal on click outside
window.addEventListener("click", (e) => {
  if (detailsModal && e.target === detailsModal) closeDetails();
  if (issueModal && e.target === issueModal) {
    pendingIssueDeliveryId = null;
    issueModal.hidden = true;
  }
});

// Close modal on Escape key
window.addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    closeDetails();
    if (issueModal) {
      pendingIssueDeliveryId = null;
      issueModal.hidden = true;
    }
  }
});

// Filter tabs
filterTabs.forEach((tab) => {
  tab.addEventListener("click", (e) => {
    filterTabs.forEach((t) => {
      t.className = "filter-tab rounded-full bg-stone-100 px-3.5 py-1.5 text-xs font-bold text-stone-700 hover:bg-stone-200";
    });
    e.currentTarget.className = "filter-tab rounded-full bg-emerald-800 px-3.5 py-1.5 text-xs font-bold text-white";
    currentFilter = e.currentTarget.dataset.filter || "all";
    renderUI();
  });
});

// Search input
if (searchInput) {
  searchInput.addEventListener("input", (e) => {
    currentSearch = e.target.value;
    renderUI();
  });
}

// Refresh button
if (refreshBtn) {
  refreshBtn.addEventListener("click", async () => {
    refreshBtn.disabled = true;
    refreshBtn.textContent = "Refreshing…";
    await loadDeliveries();
    refreshBtn.disabled = false;
    refreshBtn.textContent = "Refresh";
  });
}

// Logout button
if (logoutBtn) {
  logoutBtn.addEventListener("click", async (e) => {
    e.currentTarget.disabled = true;
    try {
      await api("/api/auth/logout", { method: "POST" });
      window.location.replace("/login");
    } catch {
      window.location.replace("/login");
    }
  });
}

async function loadDeliveries() {
  try {
    const data = await api("/api/rider/deliveries");
    allDeliveries = Array.isArray(data.deliveries) ? data.deliveries : [];
    renderUI();
  } catch (err) {
    showError(err.message || "Failed to load assigned deliveries.");
  }
}

async function init() {
  try {
    const [meResult, profileResult] = await Promise.all([
      api("/api/auth/me"),
      api("/api/rider/registration").catch(() => null),
    ]);

    const user = meResult.user;
    if (welcomeHeading) {
      welcomeHeading.textContent = `Welcome, ${user.name}.`;
    }

    if (riderStatusLine) {
      const city = profileResult?.rider?.city || "Kenya";
      const status = profileResult?.rider?.status || "approved";
      riderStatusLine.textContent = `Approved Delivery Partner · ${city} · Status: ${status}`;
    }

    await loadDeliveries();
  } catch (err) {
    showError(err.message || "Unable to initialize rider dashboard.");
  }
}

init();
