    (function () {
      function showState(id) {
        var all = document.querySelectorAll('.state');
        for (var i = 0; i < all.length; i++) all[i].classList.remove('active');
        var el = document.getElementById('state-' + id);
        if (el) el.classList.add('active');
      }

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

      function wireCodeInput(input) {
        if (!input) return;
        input.addEventListener('input', function () {
          var formatted = formatCode(input.value);
          if (formatted !== input.value) input.value = formatted;
        });
        input.addEventListener('paste', function (e) {
          e.preventDefault();
          var text = (e.clipboardData || window.clipboardData).getData('text') || '';
          input.value = formatCode(text);
        });
      }

      var params = new URLSearchParams(window.location.search);
      var code = params.get('code');
      var err = params.get('error');

      if (err === 'expired') {
        showState('c');
      } else if (err === 'invalid') {
        showState('d');
      } else if (err === 'session-gone') {
        showState('e');
      } else if (code) {
        var formatted = formatCode(code);
        document.getElementById('state-a-code').value = formatted;
        document.getElementById('state-a-code-display').textContent = formatted;
        showState('a');
      } else {
        showState('b');
        wireCodeInput(document.getElementById('code-b'));
      }
    })();
