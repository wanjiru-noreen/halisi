(() => {
  "use strict";

  const page = document.querySelector("[data-onboarding-page]");
  if (!page) return;

  const storageKey = "halisi:customer-onboarding:v1";
  const destination = page.dataset.onboardingDestination || "/customer/vendors";
  const query = new URLSearchParams(window.location.search);
  const replay = query.get("replay") === "1";

  function hasCompletedOnboarding() {
    try {
      return window.localStorage.getItem(storageKey) !== null;
    } catch {
      return false;
    }
  }

  function rememberCompletion(action) {
    try {
      window.localStorage.setItem(storageKey, JSON.stringify({ action, completed_at: new Date().toISOString() }));
    } catch {
      // Storage can be unavailable in private browsing; navigation must still work.
    }
  }

  if (hasCompletedOnboarding() && !replay) {
    window.location.replace(destination);
    return;
  }

  const slides = Array.from(document.querySelectorAll("[data-onboarding-slide]"));
  const dots = Array.from(document.querySelectorAll("[data-onboarding-dot]"));
  const back = document.getElementById("onboarding-back");
  const next = document.getElementById("onboarding-next");
  const nextLabel = document.getElementById("onboarding-next-label");
  const skip = document.getElementById("onboarding-skip");
  const status = document.getElementById("onboarding-status");
  const titles = ["Trusted Suppliers", "Fast Delivery", "Easy & Secure"];
  let current = 0;
  let touchStartX = null;

  function showSlide(index, announce = true) {
    current = Math.max(0, Math.min(index, slides.length - 1));
    slides.forEach((slide, slideIndex) => {
      const active = slideIndex === current;
      slide.dataset.state = active ? "active" : slideIndex < current ? "before" : "after";
      slide.setAttribute("aria-hidden", String(!active));
      slide.inert = !active;
    });
    dots.forEach((dot, dotIndex) => {
      if (dotIndex === current) dot.setAttribute("aria-current", "step");
      else dot.removeAttribute("aria-current");
    });
    back.hidden = current === 0;
    nextLabel.textContent = current === slides.length - 1 ? "Get Started" : "Next";
    if (announce) status.textContent = `Slide ${current + 1} of ${slides.length}: ${titles[current]}`;
  }

  function finish(action) {
    rememberCompletion(action);
    window.location.assign(destination);
  }

  skip.addEventListener("click", () => finish("skipped"));
  back.addEventListener("click", () => showSlide(current - 1));
  next.addEventListener("click", () => {
    if (current === slides.length - 1) finish("completed");
    else showSlide(current + 1);
  });
  dots.forEach((dot, index) => dot.addEventListener("click", () => showSlide(index)));

  document.addEventListener("keydown", (event) => {
    if (event.key === "ArrowLeft" && current > 0) showSlide(current - 1);
    if (event.key === "ArrowRight" && current < slides.length - 1) showSlide(current + 1);
    if (event.key === "Escape") finish("skipped");
  });

  page.addEventListener("touchstart", (event) => {
    touchStartX = event.changedTouches[0]?.clientX ?? null;
  }, { passive: true });
  page.addEventListener("touchend", (event) => {
    if (touchStartX === null) return;
    const distance = (event.changedTouches[0]?.clientX ?? touchStartX) - touchStartX;
    touchStartX = null;
    if (Math.abs(distance) < 50) return;
    if (distance < 0 && current < slides.length - 1) showSlide(current + 1);
    if (distance > 0 && current > 0) showSlide(current - 1);
  }, { passive: true });

  const requestedSlide = Number.parseInt(query.get("slide"), 10);
  const initialSlide = Number.isInteger(requestedSlide) ? requestedSlide - 1 : 0;
  showSlide(initialSlide, false);
})();
