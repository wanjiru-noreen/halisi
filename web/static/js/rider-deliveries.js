"use strict";

const errorBanner = document.getElementById("error-banner");
const successBanner = document.getElementById("success-banner");
const deliveriesList = document.getElementById("deliveries-list");
const searchInput = document.getElementById("search-input");
const filterTabs = document.querySelectorAll(".filter-tab");
const refreshBtn = document.getElementById("refresh-button");
const logoutBtn = document.getElementById("logout");

const countAll = document.getElementById("count-all");
const countActive = document.getElementById("count-active");
const countDelivered = document.getElementById("count-delivered");
const countFailed = document.getElementById("count-failed");

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

function renderCard(delivery) {
  const card = document.createElement("article");
  card.className = "rounded-2xl border border-stone-200 bg-white p-5 shadow-sm transition hover:border-emerald-300 sm:p-6";

  const top = document.createElement("div");
  top.className = "flex flex-wrap items-center justify-between gap-3 border-b border-stone-100 pb-3";

  const titleGroup = document.createElement("div");
  const title = document.createElement("h3");
  title.className = "font-bold text-stone-900 text-lg";
  title.textContent = `Delivery #${delivery.id} · Order #${delivery.order_id}`;
  const date = document.createElement("p");
  date.className = "mt-0.5 text-xs text-stone-500";
  date.textContent = delivery.delivered_at
    ? `Delivered: ${new Date(delivery.delivered_at).toLocaleString()}`
    : delivery.created_at
      ? `Assigned: ${new Date(delivery.created_at).toLocaleString()}`
      : "";
  titleGroup.append(title, date);

  top.append(titleGroup, createBadge(delivery.status));
  card.append(top);

  const grid = document.createElement("div");
  grid.className = "mt-4 grid gap-3 sm:grid-cols-2 text-sm";

  const pickupCol = document.createElement("div");
  pickupCol.className = "rounded-xl bg-stone-50 p-3";
  const pickupLabel = document.createElement("p");
  pickupLabel.className = "text-xs font-bold uppercase tracking-wider text-stone-500";
  pickupLabel.textContent = "Pickup (Vendor)";
  const pickupText = document.createElement("p");
  pickupText.className = "mt-1 font-medium text-stone-900";
  pickupText.textContent = delivery.pickup_address || "Vendor address";
  const pickupLink = document.createElement("a");
  pickupLink.href = mapUrl(delivery.pickup_address);
  pickupLink.target = "_blank";
  pickupLink.rel = "noopener noreferrer";
  pickupLink.className = "mt-2 inline-block text-xs font-semibold text-emerald-800 hover:underline";
  pickupLink.textContent = "Directions ↗";
  pickupCol.append(pickupLabel, pickupText, pickupLink);

  const dropoffCol = document.createElement("div");
  dropoffCol.className = "rounded-xl bg-stone-50 p-3";
  const dropoffLabel = document.createElement("p");
  dropoffLabel.className = "text-xs font-bold uppercase tracking-wider text-stone-500";
  dropoffLabel.textContent = "Dropoff (Customer)";
  const dropoffText = document.createElement("p");
  dropoffText.className = "mt-1 font-medium text-stone-900";
  dropoffText.textContent = delivery.delivery_address || "Customer address";
  const dropoffLink = document.createElement("a");
  dropoffLink.href = mapUrl(delivery.delivery_address);
  dropoffLink.target = "_blank";
  dropoffLink.rel = "noopener noreferrer";
  dropoffLink.className = "mt-2 inline-block text-xs font-semibold text-emerald-800 hover:underline";
  dropoffLink.textContent = "Directions ↗";
  dropoffCol.append(dropoffLabel, dropoffText, dropoffLink);

  grid.append(pickupCol, dropoffCol);
  card.append(grid);

  if (delivery.phone) {
    const contactRow = document.createElement("div");
    contactRow.className = "mt-3 flex items-center justify-between rounded-xl bg-stone-100 px-3 py-2 text-sm";
    const contactLabel = document.createElement("span");
    contactLabel.className = "text-xs font-semibold text-stone-600";
    contactLabel.textContent = "Customer Contact:";
    const contactLink = document.createElement("a");
    contactLink.href = `tel:${encodeURIComponent(delivery.phone)}`;
    contactLink.className = "font-bold text-emerald-800 hover:underline";
    contactLink.textContent = `📞 ${delivery.phone}`;
    contactRow.append(contactLabel, contactLink);
    card.append(contactRow);
  }

  const actions = document.createElement("div");
  actions.className = "mt-4 flex flex-wrap items-center justify-end gap-2 border-t border-stone-100 pt-3";

  const detailsBtn = document.createElement("button");
  detailsBtn.className = "rounded-full border border-stone-300 bg-white px-4 py-2 text-xs font-semibold text-stone-700 hover:bg-stone-50";
  detailsBtn.textContent = "View Details";
  detailsBtn.addEventListener("click", () => openDetails(delivery));
  actions.append(detailsBtn);

  if (delivery.status === "assigned") {
    const pickupBtn = document.createElement("button");
    pickupBtn.className = "rounded-full bg-emerald-800 px-4 py-2 text-xs font-bold text-white hover:bg-emerald-900";
    pickupBtn.textContent = "Confirm Gas Pickup";
    pickupBtn.addEventListener("click", () => handleStatusChange(delivery.id, "picked_up", pickupBtn));

    const issueBtn = document.createElement("button");
    issueBtn.className = "rounded-full border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 hover:bg-red-100";
    issueBtn.textContent = "Report Issue";
    issueBtn.addEventListener("click", () => promptIssue(delivery.id));

    actions.append(issueBtn, pickupBtn);
  } else if (delivery.status === "picked_up") {
    const deliverBtn = document.createElement("button");
    deliverBtn.className = "rounded-full bg-emerald-800 px-4 py-2 text-xs font-bold text-white hover:bg-emerald-900";
    deliverBtn.textContent = "Confirm Gas Delivered";
    deliverBtn.addEventListener("click", () => handleStatusChange(delivery.id, "delivered", deliverBtn));

    const issueBtn = document.createElement("button");
    issueBtn.className = "rounded-full border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700 hover:bg-red-100";
    issueBtn.textContent = "Report Issue";
    issueBtn.addEventListener("click", () => promptIssue(delivery.id));

    actions.append(issueBtn, deliverBtn);
  }

  card.append(actions);
  return card;
}

function updateCounts(deliveries) {
  const allCount = deliveries.length;
  const activeCount = deliveries.filter((d) => d.status === "assigned" || d.status === "picked_up").length;
  const deliveredCount = deliveries.filter((d) => d.status === "delivered").length;
  const failedCount = deliveries.filter((d) => d.status === "failed" || d.status === "cancelled").length;

  if (countAll) countAll.textContent = String(allCount);
  if (countActive) countActive.textContent = String(activeCount);
  if (countDelivered) countDelivered.textContent = String(deliveredCount);
  if (countFailed) countFailed.textContent = String(failedCount);
}

function renderUI() {
  updateCounts(allDeliveries);
  if (!deliveriesList) return;
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
    empty.className = "rounded-2xl border border-dashed border-stone-200 p-8 text-center text-sm text-stone-500";
    empty.textContent = allDeliveries.length === 0
      ? "No deliveries assigned to your account yet."
      : "No deliveries match your current filters.";
    deliveriesList.append(empty);
  } else {
    filtered.forEach((d) => deliveriesList.append(renderCard(d)));
    if (hasMore && !currentSearch.trim() && currentFilter === "all") {
      const loadMoreContainer = document.createElement("div");
      loadMoreContainer.className = "pt-2 text-center";
      const loadMoreBtn = document.createElement("button");
      loadMoreBtn.type = "button";
      loadMoreBtn.className = "w-full rounded-2xl border border-stone-300 bg-white py-3 text-sm font-semibold text-stone-700 hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed";
      loadMoreBtn.textContent = "Load More Deliveries";
      loadMoreBtn.onclick = (e) => loadDeliveries(false, e.currentTarget);
      loadMoreContainer.append(loadMoreBtn);
      deliveriesList.append(loadMoreContainer);
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

    showSuccess(`Delivery #${deliveryId} status updated to ${newStatus.replaceAll("_", " ")}.`);
    renderUI();

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

window.addEventListener("click", (e) => {
  if (detailsModal && e.target === detailsModal) closeDetails();
  if (issueModal && e.target === issueModal) {
    pendingIssueDeliveryId = null;
    issueModal.hidden = true;
  }
});

window.addEventListener("keydown", (e) => {
  if (e.key === "Escape") {
    closeDetails();
    if (issueModal) {
      pendingIssueDeliveryId = null;
      issueModal.hidden = true;
    }
  }
});

filterTabs.forEach((tab) => {
  tab.addEventListener("click", (e) => {
    filterTabs.forEach((t) => {
      t.className = "filter-tab rounded-full bg-stone-100 px-4 py-2 text-xs font-bold text-stone-700 hover:bg-stone-200";
    });
    e.currentTarget.className = "filter-tab rounded-full bg-emerald-800 px-4 py-2 text-xs font-bold text-white";
    currentFilter = e.currentTarget.dataset.filter || "all";
    renderUI();
  });
});

if (searchInput) {
  searchInput.addEventListener("input", (e) => {
    currentSearch = e.target.value;
    renderUI();
  });
}

if (refreshBtn) {
  refreshBtn.addEventListener("click", async () => {
    refreshBtn.disabled = true;
    refreshBtn.textContent = "Refreshing…";
    await loadDeliveries();
    refreshBtn.disabled = false;
    refreshBtn.textContent = "Refresh Deliveries";
  });
}

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

let currentOffset = 0;
let hasMore = false;
let isLoading = false;
const pageSize = 100;

async function loadDeliveries(reset = true, btn = null) {
  if (isLoading) return;
  isLoading = true;
  if (btn) {
    btn.disabled = true;
    btn.textContent = "Loading…";
  }
  try {
    if (reset) {
      currentOffset = 0;
      allDeliveries = [];
    }
    const data = await api(`/api/rider/deliveries?limit=${pageSize}&offset=${currentOffset}`);
    const fetched = Array.isArray(data.deliveries) ? data.deliveries : [];
    hasMore = fetched.length === pageSize;
    currentOffset += fetched.length;
    allDeliveries = reset ? fetched : allDeliveries.concat(fetched);
    renderUI();
  } catch (err) {
    showError(err.message || "Failed to load assigned deliveries.");
    if (btn) {
      btn.disabled = false;
      btn.textContent = "Load More Deliveries";
    }
  } finally {
    isLoading = false;
  }
}

loadDeliveries();
