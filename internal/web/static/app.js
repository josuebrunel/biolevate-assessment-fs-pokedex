// Compare tray: tracks the ticked Pokémon (at most three) and enables the
// compare button from two. The checkboxes belong to #compare-form via their
// form attribute, so this only keeps the tray in step; without JavaScript the
// form still works, it just never hides or limits itself.
(function () {
  var MAX = 3;

  function sync() {
    var boxes = Array.prototype.slice.call(document.querySelectorAll('input[name="p"]'));
    var n = boxes.filter(function (b) { return b.checked; }).length;
    boxes.forEach(function (b) { b.disabled = !b.checked && n >= MAX; });

    var tray = document.getElementById('tray');
    if (!tray) return;
    tray.hidden = n === 0;
    var text = document.getElementById('tray-text');
    var go = document.getElementById('tray-go');
    text.textContent = n === 1 ? '1 Pokémon selected. Pick one more to compare.' : n + ' Pokémon selected';
    go.disabled = n < 2;
  }

  document.addEventListener('change', function (e) {
    if (e.target && e.target.name === 'p') sync();
  });
  document.addEventListener('htmx:afterSwap', sync);
  document.addEventListener('DOMContentLoaded', sync);
})();
