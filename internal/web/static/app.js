// Party bar: tracks the ticked Pokémon (at most three), fills the slots with
// their dex numbers and enables the battle button from two. The checkboxes
// belong to #compare-form via their form attribute, so this only keeps the bar
// in step; without JavaScript the form still works, it just never hides or
// limits itself.
(function () {
  var MAX = 3;

  function sync() {
    var boxes = Array.prototype.slice.call(document.querySelectorAll('input[name="p"]'));
    var ticked = boxes.filter(function (b) { return b.checked; });
    var n = ticked.length;
    boxes.forEach(function (b) { b.disabled = !b.checked && n >= MAX; });

    var tray = document.getElementById('tray');
    if (!tray) return;
    tray.hidden = n === 0;
    Array.prototype.forEach.call(tray.querySelectorAll('.slot'), function (slot, i) {
      slot.textContent = ticked[i] ? ticked[i].getAttribute('data-dex').replace('#', '') : '';
      slot.classList.toggle('on', !!ticked[i]);
    });
    var text = document.getElementById('tray-text');
    text.textContent = n === 1 ? 'Add one more to battle' : n === MAX ? 'Party full. Ready to battle!' : 'Ready to battle!';
    document.getElementById('tray-go').disabled = n < 2;
  }

  document.addEventListener('change', function (e) {
    if (e.target && e.target.name === 'p') sync();
  });
  document.addEventListener('htmx:afterSwap', sync);
  document.addEventListener('DOMContentLoaded', sync);
})();

// Result tabs: the server renders every group stacked and the tab bar hidden.
// This turns them into tabs (one panel visible), with arrow, Home and End keys.
(function () {
  function select(bar, tab) {
    var tabs = Array.prototype.slice.call(bar.querySelectorAll('[role="tab"]'));
    tabs.forEach(function (t) {
      var on = t === tab;
      t.setAttribute('aria-selected', on ? 'true' : 'false');
      t.tabIndex = on ? 0 : -1;
      document.getElementById(t.getAttribute('aria-controls')).hidden = !on;
    });
  }

  function init() {
    var bar = document.querySelector('[data-tabs]');
    if (!bar) return;
    bar.hidden = false;
    select(bar, bar.querySelector('[aria-selected="true"]') || bar.querySelector('[role="tab"]'));
  }

  document.addEventListener('click', function (e) {
    var tab = e.target.closest && e.target.closest('[role="tab"]');
    if (tab) select(tab.parentNode, tab);

    // "Also found" and "Next" links: switch to that tab and bring the bar into view.
    var go = e.target.closest && e.target.closest('[data-goto]');
    var target = go && document.getElementById(go.getAttribute('data-goto'));
    if (target && !target.parentNode.hidden) {
      e.preventDefault();
      select(target.parentNode, target);
      var calm = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      target.parentNode.scrollIntoView({ block: 'start', behavior: calm ? 'auto' : 'smooth' });
      target.focus({ preventScroll: true });
    }
  });
  document.addEventListener('keydown', function (e) {
    var tab = e.target.closest && e.target.closest('[role="tab"]');
    if (!tab) return;
    var tabs = Array.prototype.slice.call(tab.parentNode.querySelectorAll('[role="tab"]'));
    var i = tabs.indexOf(tab);
    var next = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: tabs.length - 1 }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    var t = tabs[(next + tabs.length) % tabs.length];
    select(tab.parentNode, t);
    t.focus();
  });
  document.addEventListener('htmx:afterSwap', init);
  document.addEventListener('DOMContentLoaded', init);
})();
