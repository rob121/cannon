(function () {
  var configEl = document.getElementById('cannon-realtime-config');
  if (!configEl || typeof Centrifuge === 'undefined') {
    return;
  }

  var config;
  try {
    config = JSON.parse(configEl.textContent || '{}');
  } catch (err) {
    return;
  }
  if (!config.endpoint || !config.presence) {
    return;
  }

  var currentPath = window.location.pathname || '/';
  var centrifuge = new Centrifuge(config.endpoint, {});
  var sub = centrifuge.newSubscription(config.presence, {
    data: { page: currentPath }
  });

  function updatePage(path) {
    currentPath = path || '/';
    if (sub && typeof sub.setData === 'function') {
      sub.setData({ page: currentPath });
    }
  }

  sub.subscribe();
  sub.on('subscribed', function () {
    updatePage(window.location.pathname || '/');
  });
  centrifuge.connect();

  document.addEventListener('turbo:load', function () {
    updatePage(window.location.pathname || '/');
  });
  window.addEventListener('popstate', function () {
    updatePage(window.location.pathname || '/');
  });
})();
