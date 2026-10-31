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
