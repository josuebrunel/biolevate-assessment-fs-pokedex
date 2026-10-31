// Theme: light by default, dark when the OS asks for it, and the header button
// overrides either way. Loaded in <head> without defer so the saved choice is
// applied before first paint (no flash of the wrong theme).
(function () {
  var KEY = 'theme';
  var root = document.documentElement;
  var dark = window.matchMedia('(prefers-color-scheme: dark)');

  try {
    var saved = localStorage.getItem(KEY);
    if (saved === 'light' || saved === 'dark') root.setAttribute('data-theme', saved);
  } catch (e) { /* storage blocked: follow the OS */ }

  function current() {
    return root.getAttribute('data-theme') || (dark.matches ? 'dark' : 'light');
  }

  function label(btn) {
    btn.setAttribute('aria-label', current() === 'dark' ? 'Switch to the light theme' : 'Switch to the dark theme');
  }

  document.addEventListener('click', function (e) {
    var btn = e.target.closest && e.target.closest('.theme');
    if (!btn) return;
    var next = current() === 'dark' ? 'light' : 'dark';
    root.setAttribute('data-theme', next);
    try { localStorage.setItem(KEY, next); } catch (err) { /* not persisted */ }
    label(btn);
  });
  document.addEventListener('DOMContentLoaded', function () {
    var btn = document.querySelector('.theme');
    if (btn) label(btn);
  });
})();
