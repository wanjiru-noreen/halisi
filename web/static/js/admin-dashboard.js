"use strict";

(() => {
  const byID = (id) => document.getElementById(id);
  const counts = new Intl.NumberFormat("en-KE");
  const dates = new Intl.DateTimeFormat("en-KE", { timeZone: "Africa/Nairobi", dateStyle: "medium", timeStyle: "short" });
  let activeQuery = "";
  let controller;
  let requestNumber = 0;

  // Keep PostgreSQL decimal strings exact, including amounts beyond JS's safe integer range.
  function money(value) {
    const [whole, fraction = ""] = String(value).split(".");
    return `${counts.format(BigInt(whole))}.${fraction.padEnd(2, "0")}`;
  }
  function text(id, value) { byID(id).textContent = value; }
  function node(tag, className, value) {
    const element = document.createElement(tag);
    element.className = className;
    if (value !== undefined) element.textContent = value;
    return element;
  }
  function showError(message) { text("error", message); byID("error").hidden = false; }

  function render(data) {
    const { customers, vendors, riders } = data.accounts;
    text("customers-total", counts.format(customers.total));
    text("customers-added", `${counts.format(customers.added)} new in this period`);
    text("vendors-total", counts.format(vendors.total));
    text("vendors-detail", `${counts.format(vendors.approved)} approved & active · ${counts.format(vendors.pending)} pending · ${counts.format(vendors.added)} new`);
    text("riders-total", counts.format(riders.total));
    text("riders-detail", `${counts.format(riders.approved)} approved riders · ${counts.format(riders.pending)} pending · ${counts.format(riders.added)} new`);
    text("orders-total", counts.format(data.orders.total));
    text("financial-basis", data.financial.basis);
    text("financial-unavailable", data.financial.unavailable_reason);
    text("commissions", data.financial.commissions === null ? "Not tracked" : data.financial.commissions);
    text("settlements", data.financial.settlements === null ? "Not tracked" : data.financial.settlements);
    const financial = byID("financial-rows");
    financial.replaceChildren();
    for (const currency of data.financial.currencies) {
      const row = node("tr", "text-stone-700");
      row.append(node("th", "px-5 py-5 font-semibold sm:px-6", currency.currency));
      row.firstChild.scope = "row";
      for (const key of ["gross_sales", "refunds", "net_collections"]) {
        row.append(node("td", "whitespace-nowrap px-5 py-5 tabular-nums", money(currency[key])));
      }
      financial.append(row);
    }
    byID("financial-empty").hidden = data.financial.currencies.length > 0;
    const statuses = byID("order-statuses");
    statuses.replaceChildren();
    for (const status of data.orders.by_status) {
      const group = node("div", "");
      const label = node("div", "mb-2 flex items-center justify-between gap-4 text-sm");
      label.append(node("span", "capitalize text-stone-600", status.status.replaceAll("_", " ")), node("span", "font-semibold tabular-nums", counts.format(status.count)));
      const bar = document.createElement("progress");
      bar.className = "block h-2 w-full overflow-hidden rounded-full accent-emerald-700";
      bar.max = Math.max(1, data.orders.total);
      bar.value = status.count;
      bar.setAttribute("aria-label", `${status.status}: ${status.count} of ${data.orders.total} orders`);
      group.append(label, bar);
      statuses.append(group);
    }
    byID("orders-empty").hidden = data.orders.total !== 0;
    const activity = byID("activity");
    activity.replaceChildren();
    const eventLabels = { customer_registered: "Customer registered", vendor_registered: "Vendor registered", rider_applied: "Rider application", order_created: "Order created", payment_created: "Payment recorded" };
    for (const event of data.activity) {
      const row = node("li", "flex flex-wrap items-start justify-between gap-2 py-4");
      const detail = node("div", "min-w-0");
      detail.append(node("p", "text-sm font-medium", `${eventLabels[event.event_type] || "Platform record"} #${event.resource_id}`), node("p", "mt-1 text-xs capitalize text-stone-500", event.status.replaceAll("_", " ")));
      const time = node("time", "text-xs text-stone-500", dates.format(new Date(event.occurred_at)));
      time.dateTime = event.occurred_at;
      row.append(detail, time);
      activity.append(row);
    }
    byID("activity-empty").hidden = data.activity.length > 0;
    text("updated", `Updated ${dates.format(new Date(data.generated_at))} EAT · Refreshes every minute`);
    byID("start").value = data.period.start;
    byID("end").value = data.period.end;
    activeQuery = new URLSearchParams({ start: data.period.start, end: data.period.end }).toString();
  }

  async function load(query = activeQuery) {
    controller?.abort();
    controller = new AbortController();
    const current = ++requestNumber;
    byID("error").hidden = true;
    byID("dashboard-data").hidden = true;
    byID("loading").hidden = false;
    byID("apply").disabled = true;
    try {
      const response = await fetch(`/api/admin/dashboard${query ? `?${query}` : ""}`, { signal: controller.signal, credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
      if (response.status === 401) { window.location.replace("/login"); return; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Could not load platform statistics.");
      if (current !== requestNumber) return;
      render(data);
      byID("dashboard-data").hidden = false;
    } catch (error) {
      if (current === requestNumber && error.name !== "AbortError") {
        text("updated", "Statistics unavailable. Apply dates to retry.");
        showError(error.message);
      }
    } finally {
      if (current === requestNumber) { byID("loading").hidden = true; byID("apply").disabled = false; }
    }
  }
  byID("filters").addEventListener("submit", (event) => {
    event.preventDefault();
    load(new URLSearchParams({ start: byID("start").value, end: byID("end").value }).toString());
  });
  byID("last-week").addEventListener("click", () => {
    const last = new Date(Date.now() + 3 * 60 * 60 * 1000);
    const first = new Date(last);
    first.setUTCDate(first.getUTCDate() - 6);
    load(new URLSearchParams({ start: first.toISOString().slice(0, 10), end: last.toISOString().slice(0, 10) }).toString());
  });
  byID("logout").addEventListener("click", async (event) => {
    event.currentTarget.disabled = true;
    try {
      const response = await fetch("/api/auth/logout", { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json" }, body: "{}" });
      if (!response.ok) throw new Error("Could not log out. Please try again.");
      controller?.abort();
      byID("dashboard-data").hidden = true;
      window.location.replace("/login");
    } catch (error) { showError(error.message); event.currentTarget.disabled = false; }
  });
  setInterval(() => { if (!document.hidden) load(); }, 60000);
  load();
})();
