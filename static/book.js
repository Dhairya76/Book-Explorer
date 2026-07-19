// Progressive enhancement: type the first sentence of the summary like ink
// being written, then reveal the rest. If JS is off, the full text is already
// in the page. If the user prefers reduced motion, we leave it untouched.
(function () {
  var el = document.querySelector(".book-text[data-typewriter]");
  if (!el) return;

  if (window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    return;
  }

  var full = el.textContent;
  if (!full) return;

  // First sentence, ending at a period/!/? that follows a lowercase letter or
  // digit — this skips abbreviation dots like "J.R.R.". Fall back to a chunk.
  var match = full.match(/[a-z0-9)"'][.!?](\s|$)/);
  var splitAt = match ? match.index + 2 : Math.min(90, full.length);
  // Don't let the first line run too long before revealing the rest.
  if (splitAt > 180) {
    var slice = full.slice(0, 180);
    var lastSpace = slice.lastIndexOf(" ");
    splitAt = lastSpace > 60 ? lastSpace : 180;
  }
  var firstLine = full.slice(0, splitAt);
  var rest = full.slice(splitAt);

  var typed = document.createElement("span");
  var caret = document.createElement("span");
  caret.className = "type-caret";
  caret.innerHTML = "&nbsp;";
  var restSpan = document.createElement("span");
  restSpan.textContent = rest;
  restSpan.style.opacity = "0";
  restSpan.style.transition = "opacity 0.6s ease";

  el.textContent = "";
  el.appendChild(typed);
  el.appendChild(caret);
  el.appendChild(restSpan);

  var i = 0;
  var START_DELAY = 1700; // let the cover finish flipping open first
  var SPEED = 18; // ms per character

  function tick() {
    typed.textContent = firstLine.slice(0, i);
    i += 1;
    if (i <= firstLine.length) {
      setTimeout(tick, SPEED);
    } else {
      caret.remove();
      restSpan.style.opacity = "1";
    }
  }

  setTimeout(tick, START_DELAY);
})();

// Cover reliability: Open Library sometimes serves a broken/redirected image
// for a given cover id. If the current one errors, try the book's other cover
// ids in turn; if all fail, hide the image and let the title cover show through.
(function () {
  document.querySelectorAll(".cover-art[data-fallbacks]").forEach(function (img) {
    var ids = img.getAttribute("data-fallbacks").split(",").filter(Boolean);
    var idx = 0;
    img.addEventListener("error", function () {
      idx += 1;
      if (idx < ids.length) {
        img.src = "https://covers.openlibrary.org/b/id/" + ids[idx] + "-L.jpg?default=false";
      } else {
        img.style.display = "none";
      }
    });
  });
})();
