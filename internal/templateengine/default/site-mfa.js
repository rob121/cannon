(function () {
  var busy = false;

  function b64ToBuf(b64) {
    var pad = '='.repeat((4 - (b64.length % 4)) % 4);
    var bin = atob((b64 + pad).replace(/-/g, '+').replace(/_/g, '/'));
    var bytes = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    return bytes.buffer;
  }

  function bufToB64(buf) {
    var bytes = new Uint8Array(buf);
    var bin = '';
    for (var i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i]);
    return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  function prepGetOptions(options) {
    options.challenge = b64ToBuf(options.challenge);
    if (options.allowCredentials) {
      options.allowCredentials = options.allowCredentials.map(function (c) {
        return Object.assign({}, c, { id: b64ToBuf(c.id) });
      });
    }
    return options;
  }

  function prepCreateOptions(options) {
    options.challenge = b64ToBuf(options.challenge);
    if (typeof options.user.id === 'string') {
      options.user.id = new TextEncoder().encode(options.user.id);
    } else {
      options.user.id = b64ToBuf(options.user.id);
    }
    if (options.excludeCredentials) {
      options.excludeCredentials = options.excludeCredentials.map(function (c) {
        return Object.assign({}, c, { id: b64ToBuf(c.id) });
      });
    }
    return options;
  }

  function showError(id, msg) {
    var el = document.getElementById(id);
    if (!el) return;
    el.textContent = msg;
    el.classList.remove('d-none');
  }

  function resolveCSRF(explicit) {
    if (explicit) return explicit;
    var el = document.querySelector('input[name="_csrf"]');
    return el ? el.value : '';
  }

  function csrfHeaders(token, extra) {
    var headers = Object.assign({}, extra || {});
    if (token) headers['X-CSRF-Token'] = token;
    return headers;
  }

  async function readJSON(response) {
    var text = await response.text();
    var trimmed = (text || '').trim();
    if (!trimmed) return {};
    try {
      return JSON.parse(trimmed);
    } catch (err) {
      throw new Error(trimmed || ('HTTP ' + response.status));
    }
  }

  function trimmedError(payload) {
    if (typeof payload === 'string') return payload;
    return '';
  }

  window.cannonPasskeyAuth = async function (beginURL, finishURL, errorID, csrf) {
    if (busy) return;
    busy = true;
    try {
      var token = resolveCSRF(csrf);
      var begin = await fetch(beginURL, {
        method: 'POST',
        credentials: 'same-origin',
        headers: csrfHeaders(token)
      });
      var payload = await readJSON(begin);
      if (!begin.ok) throw new Error(payload.error || trimmedError(payload) || 'Could not start passkey sign-in');
      var options = prepGetOptions(payload.publicKey);
      var cred = await navigator.credentials.get({ publicKey: options });
      var body = {
        id: cred.id,
        rawId: bufToB64(cred.rawId),
        type: cred.type,
        response: {
          authenticatorData: bufToB64(cred.response.authenticatorData),
          clientDataJSON: bufToB64(cred.response.clientDataJSON),
          signature: bufToB64(cred.response.signature),
          userHandle: cred.response.userHandle ? bufToB64(cred.response.userHandle) : null
        }
      };
      var finish = await fetch(finishURL, {
        method: 'POST',
        credentials: 'same-origin',
        headers: csrfHeaders(token, { 'Content-Type': 'application/json' }),
        body: JSON.stringify(body)
      });
      var result = await readJSON(finish);
      if (!finish.ok) throw new Error(result.error || trimmedError(result) || 'Passkey verification failed');
      if (result.redirect) window.location.href = result.redirect;
    } catch (err) {
      showError(errorID, err.message || 'Passkey sign-in failed');
    } finally {
      busy = false;
    }
  };

  window.cannonPasskeyRegister = async function (beginURL, finishURL, name, errorID, csrf) {
    if (busy) return;
    busy = true;
    try {
      var token = resolveCSRF(csrf);
      var begin = await fetch(beginURL, {
        method: 'POST',
        credentials: 'same-origin',
        headers: csrfHeaders(token)
      });
      var payload = await readJSON(begin);
      if (!begin.ok) throw new Error(payload.error || trimmedError(payload) || 'Could not start passkey registration');
      var options = prepCreateOptions(payload.publicKey);
      var cred = await navigator.credentials.create({ publicKey: options });
      var body = {
        id: cred.id,
        rawId: bufToB64(cred.rawId),
        type: cred.type,
        response: {
          clientDataJSON: bufToB64(cred.response.clientDataJSON),
          attestationObject: bufToB64(cred.response.attestationObject)
        }
      };
      var finish = await fetch(finishURL, {
        method: 'POST',
        credentials: 'same-origin',
        headers: csrfHeaders(token, { 'Content-Type': 'application/json', 'X-Passkey-Name': name }),
        body: JSON.stringify(body)
      });
      var result = await readJSON(finish);
      if (!finish.ok) throw new Error(result.error || trimmedError(result) || 'Passkey registration failed');
      if (result.redirect) window.location.href = result.redirect;
    } catch (err) {
      showError(errorID, err.message || 'Passkey registration failed');
    } finally {
      busy = false;
    }
  };

  // Idempotent binders so duplicate login/security partials do not stack listeners.
  window.cannonBindPasskeyAuth = function (btn, errorID) {
    if (!btn || btn.dataset.passkeyBound === '1') return;
    btn.dataset.passkeyBound = '1';
    btn.addEventListener('click', function () {
      cannonPasskeyAuth(btn.dataset.begin, btn.dataset.finish, errorID || 'login-passkey-error', btn.dataset.csrf);
    });
  };

  window.cannonBindPasskeyRegister = function (btn, errorID) {
    if (!btn || btn.dataset.passkeyBound === '1') return;
    btn.dataset.passkeyBound = '1';
    btn.addEventListener('click', function () {
      var name = document.getElementById('passkey-name')?.value || 'Passkey';
      cannonPasskeyRegister(btn.dataset.begin, btn.dataset.finish, name, errorID || 'passkey-register-error', btn.dataset.csrf);
    });
  };

  window.cannonBindPasskeyMFA = function (btn, errorID) {
    if (!btn || btn.dataset.passkeyBound === '1') return;
    btn.dataset.passkeyBound = '1';
    btn.addEventListener('click', function () {
      cannonPasskeyAuth(btn.dataset.begin, btn.dataset.finish, errorID || 'mfa-passkey-error', btn.dataset.csrf);
    });
  };
})();
