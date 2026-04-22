    (function () {
      var input = document.getElementById('code');
      if (!input) return;

      // Strip everything outside the base32 alphabet (A-Z, 2-7), force uppercase,
      // then reinsert a single dash after the 4th character. Max 8 alphanumeric
      // chars + 1 dash = 9 visible chars.
      function formatCode(raw) {
        var upper = (raw || '').toUpperCase();
        var cleaned = '';
        for (var i = 0; i < upper.length && cleaned.length < 8; i++) {
          var ch = upper.charAt(i);
          if ((ch >= 'A' && ch <= 'Z') || (ch >= '2' && ch <= '7')) {
            cleaned += ch;
          }
        }
        if (cleaned.length <= 4) return cleaned;
        return cleaned.slice(0, 4) + '-' + cleaned.slice(4);
      }

      input.addEventListener('input', function () {
        var formatted = formatCode(input.value);
        if (formatted !== input.value) input.value = formatted;
      });

      input.addEventListener('paste', function (e) {
        e.preventDefault();
        var text = (e.clipboardData || window.clipboardData).getData('text') || '';
        input.value = formatCode(text);
      });
    })();
