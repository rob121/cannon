(function () {
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

  window.cannonPasskeyAuth = async function (beginURL, finishURL, errorID) {
    try {
      var begin = await fetch(beginURL, { method: 'POST', credentials: 'same-origin' });
      var payload = await begin.json();
      if (!begin.ok) throw new Error(payload.error || 'Could not start passkey sign-in');
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
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      var result = await finish.json();
      if (!finish.ok) throw new Error(result.error || 'Passkey verification failed');
      if (result.redirect) window.location.href = result.redirect;
    } catch (err) {
      showError(errorID, err.message || 'Passkey sign-in failed');
    }
  };

  window.cannonPasskeyRegister = async function (beginURL, finishURL, name, errorID) {
    try {
      var begin = await fetch(beginURL, { method: 'POST', credentials: 'same-origin' });
      var payload = await begin.json();
      if (!begin.ok) throw new Error(payload.error || 'Could not start passkey registration');
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
        headers: { 'Content-Type': 'application/json', 'X-Passkey-Name': name },
        body: JSON.stringify(body)
      });
      var result = await finish.json();
      if (!finish.ok) throw new Error(result.error || 'Passkey registration failed');
      if (result.redirect) window.location.href = result.redirect;
    } catch (err) {
      showError(errorID, err.message || 'Passkey registration failed');
    }
  };
})();
