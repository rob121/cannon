(function () {
  var STORAGE_KEY = 'cannon-admin-sidebar-collapsed';
  var NAV_GROUP_PREFIX = 'cannon-admin-nav-';

  function isCollapsed() {
    return document.body.classList.contains('admin-sidebar-collapsed');
  }

  function applySidebarCollapsed(collapsed) {
    document.body.classList.toggle('admin-sidebar-collapsed', collapsed);
    var btn = document.getElementById('admin-sidebar-toggle');
    if (!btn) {
      return;
    }
    btn.setAttribute('aria-pressed', collapsed ? 'true' : 'false');
    btn.setAttribute('title', collapsed ? 'Expand Sidebar' : 'Collapse Sidebar');
    var icon = btn.querySelector('i');
    if (icon) {
      icon.className = collapsed ? 'bi bi-chevron-right' : 'bi bi-chevron-left';
    }
  }

  function toggleSidebar() {
    var next = !isCollapsed();
    applySidebarCollapsed(next);
    try {
      localStorage.setItem(STORAGE_KEY, next ? '1' : '0');
    } catch (e) {}
  }

  function syncNavGroup(groupId) {
    var group = document.getElementById(groupId);
    if (!group) {
      return;
    }
    var toggle = group.querySelector('.admin-nav-group-toggle');
    if (group.classList.contains('open')) {
      if (toggle) {
        toggle.setAttribute('aria-expanded', 'true');
      }
      return;
    }
    try {
      if (localStorage.getItem(NAV_GROUP_PREFIX + groupId + '-open') === '1') {
        group.classList.add('open');
        if (toggle) {
          toggle.setAttribute('aria-expanded', 'true');
        }
      } else if (toggle) {
        toggle.setAttribute('aria-expanded', 'false');
      }
    } catch (e) {}
  }

  function initFormToggles() {
    document.querySelectorAll('.admin-form-toggle-input').forEach(function (input) {
      if (input.dataset.bound) {
        return;
      }
      input.dataset.bound = '1';
      var label = input.parentElement;
      var text = label ? label.querySelector('.admin-form-toggle-text') : null;
      if (!text) {
        return;
      }
      var update = function () {
        var on = text.dataset.on || 'On';
        var off = text.dataset.off || 'Off';
        text.textContent = input.checked ? on : off;
      };
      update();
      input.addEventListener('change', update);
    });
  }

  function initBlockForm() {
    var typeEl = document.getElementById('block-type');
    var nativeEl = document.getElementById('block-fields-native');
    var extEl = document.getElementById('block-fields-extension');
    var extSelect = document.getElementById('block-extension');
    var itemSelect = document.getElementById('block-extension-item');
    var dataFields = Array.prototype.slice.call(document.querySelectorAll('.block-data-field'));
    if (!typeEl || !nativeEl || !extEl || !extSelect || !itemSelect || typeEl.dataset.bound) {
      return;
    }
    typeEl.dataset.bound = '1';

    function syncBlockDataFields() {
      var name = extSelect.value;
      var item = itemSelect.value;
      dataFields.forEach(function (field) {
        var matches = field.dataset.extensionName === name && field.dataset.blockId === item;
        field.style.display = matches ? '' : 'none';
        field.querySelectorAll('input, select, textarea').forEach(function (input) {
          input.disabled = !matches;
        });
      });
    }

    function syncBlockItems() {
      var name = extSelect.value;
      var selected = itemSelect.selectedOptions.length ? itemSelect.selectedOptions[0] : null;
      var selectedMatches = selected && (!selected.dataset.extensionName || selected.dataset.extensionName === name);
      var firstMatch = null;

      itemSelect.querySelectorAll('option').forEach(function (option) {
        var optionExtension = option.dataset.extensionName || '';
        if (!optionExtension) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = !name || optionExtension === name;
        option.hidden = !matches;
        option.disabled = !matches;
        if (matches && !firstMatch) {
          firstMatch = option;
        }
      });

      if (!selectedMatches) {
        itemSelect.value = '';
      }
      if (name && !itemSelect.value && firstMatch) {
        firstMatch.selected = true;
      }
      syncBlockDataFields();
    }

    function syncTypeFields() {
      var isExtension = typeEl.value === 'extension';
      var isContent = typeEl.value === 'content';
      var contentEl = document.getElementById('block-fields-content');
      nativeEl.style.display = (isExtension || isContent) ? 'none' : '';
      if (contentEl) {
        contentEl.style.display = isContent ? '' : 'none';
      }
      extEl.style.display = isExtension ? '' : 'none';
      syncBlockItems();
    }

    typeEl.addEventListener('change', syncTypeFields);
    extSelect.addEventListener('change', syncBlockItems);
    itemSelect.addEventListener('change', syncBlockDataFields);
    syncTypeFields();
  }

  function initRouteForm() {
    var typeEl = document.getElementById('route-type');
    var targetEl = document.getElementById('route-fields-target');
    var extEl = document.getElementById('route-fields-extension');
    var endpointEl = document.getElementById('route-fields-endpoint');
    var ctrlEl = document.getElementById('route-fields-controller');
    var ctrlSelect = document.getElementById('route-controller');
    var ctrlActionSelect = document.getElementById('route-controller-action');
    var pathInput = document.querySelector('input[name="path"]');
    var extSelect = document.getElementById('route-extension');
    var pageSelect = document.getElementById('route-extension-page');
    var endpointExtSelect = document.getElementById('route-endpoint-extension');
    var endpointSelect = document.getElementById('route-extension-endpoint');
    var dataFields = Array.prototype.slice.call(document.querySelectorAll('.route-data-field'));
    var endpointDataFields = Array.prototype.slice.call(document.querySelectorAll('.route-endpoint-data-field'));
    var helpUrl = document.querySelector('.route-help-url');
    var helpLocal = document.querySelector('.route-help-local');
    if (!typeEl || !targetEl || !extEl || !endpointEl || !ctrlEl || typeEl.dataset.bound) {
      return;
    }
    typeEl.dataset.bound = '1';

    function setPanelVisible(panel, visible) {
      if (!panel) {
        return;
      }
      panel.hidden = !visible;
      panel.querySelectorAll('input, select, textarea, button').forEach(function (input) {
        if (input === typeEl || input.type === 'hidden') {
          return;
        }
        input.disabled = !visible;
      });
    }

    function syncPageDataFields() {
      var extensionVisible = typeEl.value === 'Extension' && extEl && !extEl.hidden;
      if (!extensionVisible || !extSelect || !pageSelect) {
        dataFields.forEach(function (field) {
          field.hidden = true;
          field.querySelectorAll('input, select, textarea').forEach(function (input) {
            input.disabled = true;
          });
        });
        return;
      }
      var name = extSelect.value;
      var page = pageSelect.value;
      dataFields.forEach(function (field) {
        var matches = field.dataset.extensionName === name && field.dataset.pageId === page;
        field.hidden = !matches;
        field.querySelectorAll('input, select, textarea').forEach(function (input) {
          input.disabled = !matches;
        });
      });
    }

    function syncEndpointDataFields() {
      var endpointVisible = typeEl.value === 'Extension Endpoint' && endpointEl && !endpointEl.hidden;
      if (!endpointVisible || !endpointExtSelect || !endpointSelect) {
        endpointDataFields.forEach(function (field) {
          field.hidden = true;
          field.querySelectorAll('input, select, textarea').forEach(function (input) {
            input.disabled = true;
          });
        });
        return;
      }
      var name = endpointExtSelect.value;
      var endpoint = endpointSelect.value;
      endpointDataFields.forEach(function (field) {
        var matches = field.dataset.extensionName === name && field.dataset.endpointId === endpoint;
        field.hidden = !matches;
        field.querySelectorAll('input, select, textarea').forEach(function (input) {
          input.disabled = !matches;
        });
      });
    }

    function syncPageItems() {
      if (!extSelect || !pageSelect || extEl.hidden) {
        syncPageDataFields();
        return;
      }
      var name = extSelect.value;
      var selected = pageSelect.selectedOptions.length ? pageSelect.selectedOptions[0] : null;
      var selectedMatches = selected && (!selected.dataset.extensionName || selected.dataset.extensionName === name);
      var firstMatch = null;

      pageSelect.querySelectorAll('option').forEach(function (option) {
        var optionExtension = option.dataset.extensionName || '';
        if (!optionExtension) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = !name || optionExtension === name;
        option.hidden = !matches;
        option.disabled = !matches;
        if (matches && !firstMatch) {
          firstMatch = option;
        }
      });

      if (!selectedMatches) {
        pageSelect.value = '';
      }
      if (name && !pageSelect.value && firstMatch) {
        firstMatch.selected = true;
      }
      syncPageDataFields();
    }

    function syncEndpointItems() {
      if (!endpointExtSelect || !endpointSelect || endpointEl.hidden) {
        syncEndpointDataFields();
        return;
      }
      var name = endpointExtSelect.value;
      var selected = endpointSelect.selectedOptions.length ? endpointSelect.selectedOptions[0] : null;
      var selectedMatches = selected && (!selected.dataset.extensionName || selected.dataset.extensionName === name);
      var firstMatch = null;

      endpointSelect.querySelectorAll('option').forEach(function (option) {
        var optionExtension = option.dataset.extensionName || '';
        if (!optionExtension) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = !name || optionExtension === name;
        option.hidden = !matches;
        option.disabled = !matches;
        if (matches && !firstMatch) {
          firstMatch = option;
        }
      });

      if (!selectedMatches) {
        endpointSelect.value = '';
      }
      if (name && !endpointSelect.value && firstMatch) {
        firstMatch.selected = true;
      }
      syncEndpointDataFields();
    }

    function syncControllerItems() {
      if (!ctrlSelect || !ctrlActionSelect || ctrlEl.hidden) {
        return;
      }
      var ctrlID = ctrlSelect.value;
      var selected = ctrlActionSelect.selectedOptions.length ? ctrlActionSelect.selectedOptions[0] : null;
      var selectedMatches = selected && (!selected.dataset.controllerId || selected.dataset.controllerId === ctrlID);
      var firstMatch = null;

      ctrlActionSelect.querySelectorAll('option').forEach(function (option) {
        var optionCtrl = option.dataset.controllerId || '';
        if (!optionCtrl) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = !ctrlID || optionCtrl === ctrlID;
        option.hidden = !matches;
        option.disabled = !matches;
        if (matches && !firstMatch) {
          firstMatch = option;
        }
      });

      if (!selectedMatches) {
        ctrlActionSelect.value = '';
      }
      if (ctrlID && !ctrlActionSelect.value && firstMatch) {
        firstMatch.selected = true;
      }
    }

    function syncRouteFields() {
      var type = typeEl.value;
      var showTarget = type === 'Url' || type === 'Local File';
      setPanelVisible(targetEl, showTarget);
      setPanelVisible(extEl, type === 'Extension');
      setPanelVisible(endpointEl, type === 'Extension Endpoint');
      setPanelVisible(ctrlEl, type === 'Controller');
      if (helpUrl) {
        helpUrl.hidden = type !== 'Url';
      }
      if (helpLocal) {
        helpLocal.hidden = type !== 'Local File';
      }
      if (type === 'Extension') {
        syncPageItems();
      } else {
        syncPageDataFields();
      }
      if (type === 'Extension Endpoint') {
        syncEndpointItems();
      } else {
        syncEndpointDataFields();
      }
      if (type === 'Controller') {
        syncControllerItems();
      }
    }

    typeEl.addEventListener('change', syncRouteFields);
    if (extSelect) extSelect.addEventListener('change', syncPageItems);
    if (pageSelect) pageSelect.addEventListener('change', syncPageDataFields);
    if (endpointExtSelect) endpointExtSelect.addEventListener('change', syncEndpointItems);
    if (endpointSelect) endpointSelect.addEventListener('change', syncEndpointDataFields);
    if (ctrlSelect) ctrlSelect.addEventListener('change', syncControllerItems);
    if (ctrlActionSelect) {
      ctrlActionSelect.addEventListener('change', function () {
        var selected = ctrlActionSelect.selectedOptions.length ? ctrlActionSelect.selectedOptions[0] : null;
        if (selected && selected.dataset.defaultPath && pathInput && !pathInput.value) {
          pathInput.value = selected.dataset.defaultPath;
        }
      });
    }
    syncRouteFields();
  }

  function init() {
    var collapsed = false;
    try {
      collapsed = localStorage.getItem(STORAGE_KEY) === '1';
    } catch (e) {}
    applySidebarCollapsed(collapsed);
    syncNavGroup('nav-users-group');
    syncNavGroup('nav-menus-group');
    syncNavGroup('nav-system-group');
    syncNavGroup('nav-extension-apps-group');
    initFormToggles();
    initBlockForm();
    initRouteForm();
  }

  document.addEventListener('click', function (e) {
    if (e.target.closest('#admin-sidebar-toggle')) {
      e.preventDefault();
      toggleSidebar();
      return;
    }
    if (e.target.closest('#admin-mobile-menu')) {
      e.preventDefault();
      document.body.classList.toggle('admin-sidebar-open');
      return;
    }
    var toggle = e.target.closest('.admin-nav-group-toggle');
    if (!toggle || !toggle.closest('.admin-body')) {
      return;
    }
    e.preventDefault();
    var group = toggle.closest('.admin-nav-group');
    if (!group || !group.id) {
      return;
    }
    var open = group.classList.toggle('open');
    toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    try {
      localStorage.setItem(NAV_GROUP_PREFIX + group.id + '-open', open ? '1' : '0');
    } catch (err) {}
  });

  document.addEventListener('turbo:before-cache', function () {
    document.querySelectorAll('[data-bound]').forEach(function (el) {
      delete el.dataset.bound;
    });
  });

  document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('turbo:load', init);
})();
