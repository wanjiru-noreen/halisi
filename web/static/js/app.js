// Halisi — animated stat counters
document.addEventListener('DOMContentLoaded', function () {
  var counters = document.querySelectorAll('.stat__value[data-count]');

  var animate = function (el) {
    var target = parseInt(el.getAttribute('data-count'), 10);
    var duration = 900;
    var start = null;

    function step(timestamp) {
      if (!start) start = timestamp;
      var progress = Math.min((timestamp - start) / duration, 1);
      var value = Math.round(target * progress);
      el.textContent = value.toLocaleString() + '+';
      if (progress < 1) requestAnimationFrame(step);
    }
    requestAnimationFrame(step);
  };

  if ('IntersectionObserver' in window) {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) {
          animate(entry.target);
          observer.unobserve(entry.target);
        }
      });
    }, { threshold: 0.4 });
    counters.forEach(function (el) { observer.observe(el); });
  } else {
    counters.forEach(animate);
  }
});