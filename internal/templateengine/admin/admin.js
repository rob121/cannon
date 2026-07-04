(function () {
  var STORAGE_KEY = 'cannon-admin-sidebar-collapsed';
  var NAV_GROUP_PREFIX = 'cannon-admin-nav-';
  var CKEDITOR5_URL = 'https://cdn.jsdelivr.net/npm/@ckeditor/ckeditor5-build-classic@41.4.2/build/ckeditor.js';
  var CODEMIRROR_BASE = 'https://cdn.jsdelivr.net/npm/codemirror@5.65.16';
  var ckEditorLoadPromise = null;
  var codeMirrorLoadPromise = null;
  var ckEditorInstances = [];
  var templateEditors = [];

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
    document.querySelectorAll('.admin-nav-group').forEach(function (group) {
      if (collapsed && group.classList.contains('open')) {
        positionCollapsedNavFlyout(group);
      } else {
        clearNavFlyoutPosition(group);
      }
    });
  }

  function clearNavFlyoutPosition(group) {
    var sub = group.querySelector('.admin-nav-sub');
    if (!sub) {
      return;
    }
    sub.style.top = '';
    sub.style.maxHeight = '';
    sub.style.overflowY = '';
  }

  function positionCollapsedNavFlyout(group) {
    if (!isCollapsed()) {
      return;
    }
    var toggle = group.querySelector('.admin-nav-group-toggle');
    var sub = group.querySelector('.admin-nav-sub');
    if (!toggle || !sub) {
      return;
    }
    var rect = toggle.getBoundingClientRect();
    var top = Math.max(0.5, rect.top);
    sub.style.top = top + 'px';
    sub.style.maxHeight = Math.max(120, window.innerHeight - top - 0.5) + 'px';
    sub.style.overflowY = 'auto';
  }

  function closeCollapsedNavFlyouts(exceptGroup) {
    document.querySelectorAll('.admin-nav-group.open').forEach(function (group) {
      if (exceptGroup && group === exceptGroup) {
        return;
      }
      group.classList.remove('open');
      var toggle = group.querySelector('.admin-nav-group-toggle');
      if (toggle) {
        toggle.setAttribute('aria-expanded', 'false');
      }
      clearNavFlyoutPosition(group);
    });
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

  function refreshCKEditors() {
    ckEditorInstances.forEach(function (editor) {
      if (editor && editor.ui && typeof editor.ui.update === 'function') {
        editor.ui.update();
      }
    });
    window.dispatchEvent(new Event('resize'));
  }

  function initLazyCKEditors(container) {
    if (!container) {
      return;
    }
    var pending = [];
    container.querySelectorAll('[data-ckeditor]').forEach(function (el) {
      if (!el.id || el.dataset.ckEditorReady) {
        return;
      }
      var key = el.getAttribute('data-ckeditor') || el.id;
      var config = key === 'body' ? ckEditorBodyToolbar : ckEditorIntroToolbar;
      pending.push({ element: el, config: config, key: key });
    });
    if (!pending.length) {
      return;
    }
    loadCKEditor5().then(function (ClassicEditor) {
      return Promise.all(pending.map(function (item) {
        if (item.element.dataset.ckEditorReady) {
          return null;
        }
        return ClassicEditor.create(item.element, item.config).then(function (editor) {
          item.element.dataset.ckEditorReady = '1';
          ckEditorInstances.push(editor);
          return editor;
        });
      }));
    }).then(function () {
      refreshCKEditors();
    }).catch(function (err) {
      console.error('CKEditor lazy init failed:', err);
    });
  }

  function initSidebarTabs(root) {
    var scope = root || document;
    scope.querySelectorAll('.admin-form-sidebar-card, .admin-form-tabbed-card').forEach(function (card) {
      if (card.dataset.bound) {
        return;
      }
      card.dataset.bound = '1';
      card.querySelectorAll('[data-sidebar-tab]').forEach(function (btn) {
        btn.addEventListener('click', function () {
          var tab = btn.getAttribute('data-sidebar-tab');
          card.querySelectorAll('[data-sidebar-tab]').forEach(function (item) {
            var active = item === btn;
            item.classList.toggle('is-active', active);
            item.setAttribute('aria-selected', active ? 'true' : 'false');
          });
          card.querySelectorAll('[data-sidebar-panel]').forEach(function (panel) {
            panel.classList.toggle('is-active', panel.getAttribute('data-sidebar-panel') === tab);
          });
          if (tab === 'placement') {
            initBlockRoutePicker();
          }
          if (card.closest('#item-form')) {
            if (tab === 'content' || tab === 'fields') {
              var panel = card.querySelector('[data-sidebar-panel="' + tab + '"]');
              if (tab === 'fields') {
                initLazyCKEditors(panel);
              }
              window.requestAnimationFrame(refreshCKEditors);
            }
          }
        });
      });
    });
  }

  function initFormToggles() {
    // Toggle labels are rendered as on/off spans; visibility is handled in CSS.
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
      var isLogin = typeEl.value === 'login';
      var isMenu = typeEl.value === 'menu-vertical' || typeEl.value === 'menu-horizontal';
      var isSearch = typeEl.value === 'search-horizontal' || typeEl.value === 'search-vertical';
      var contentEl = document.getElementById('block-fields-content');
      var loginEl = document.getElementById('block-fields-login');
      var menuEl = document.getElementById('block-fields-menu');
      var searchEl = document.getElementById('block-fields-search');
      nativeEl.style.display = (isExtension || isContent || isLogin || isMenu || isSearch) ? 'none' : '';
      if (contentEl) {
        contentEl.hidden = !isContent;
        contentEl.style.display = isContent ? '' : 'none';
      }
      if (loginEl) {
        loginEl.hidden = !isLogin;
        loginEl.style.display = isLogin ? '' : 'none';
      }
      if (menuEl) {
        menuEl.hidden = !isMenu;
        menuEl.style.display = isMenu ? '' : 'none';
        menuEl.querySelectorAll('select, input').forEach(function (input) {
          input.required = isMenu && input.name === 'menu_name';
        });
      }
      if (searchEl) {
        searchEl.hidden = !isSearch;
        searchEl.style.display = isSearch ? '' : 'none';
      }
      extEl.hidden = !isExtension;
      extEl.style.display = isExtension ? '' : 'none';
      syncBlockItems();

      var contentTextarea = nativeEl.querySelector('textarea.admin-template-editor');
      if (contentTextarea) {
        contentTextarea.setAttribute('data-editor-mode', typeEl.value === 'markdown' ? 'markdown' : 'htmlmixed');
      }
      var showNative = !(isExtension || isContent || isLogin || isMenu || isSearch);
      if (showNative) {
        initTemplateEditors();
        syncBlockContentEditorMode();
        refreshTemplateEditors();
      }
    }

    typeEl.addEventListener('change', syncTypeFields);
    extSelect.addEventListener('change', syncBlockItems);
    itemSelect.addEventListener('change', syncBlockDataFields);
    syncTypeFields();
  }

  function initBlockRoutePicker() {
    var panel = document.getElementById('block-route-panel');
    var modeSelect = document.getElementById('block-route-mode');
    var body = document.getElementById('block-route-body');
    var hintAll = document.getElementById('block-route-hint-all');
    var list = document.getElementById('block-route-list');
    var searchInput = document.getElementById('block-route-search');
    var selectedCount = document.getElementById('block-route-selected-count');
    if (!panel || !modeSelect || !list) {
      return;
    }

    function routeCheckboxes() {
      return Array.prototype.slice.call(list.querySelectorAll('input[name="route_ids"]'));
    }

    function updateSelectedCount() {
      if (!selectedCount) {
        return;
      }
      var count = routeCheckboxes().filter(function (input) { return input.checked; }).length;
      selectedCount.textContent = count + (count === 1 ? ' Selected' : ' Selected');
    }

    function syncRouteModeUI() {
      var mode = modeSelect.value || 'all';
      var needsSelection = mode === 'only' || mode === 'except';
      if (body) {
        body.hidden = false;
      }
      if (list) {
        list.classList.toggle('is-disabled', !needsSelection);
      }
      if (hintAll) {
        hintAll.hidden = false;
        var hintText = hintAll.querySelector('.admin-block-route-hint-text');
        if (hintText) {
          if (mode === 'none') {
            hintText.textContent = 'This block will not render on any page until you change the visibility mode.';
          } else if (needsSelection) {
            hintText.textContent = 'Select the pages where this block should appear (or be excluded). Expand each group below to choose routes.';
          } else {
            hintText.innerHTML = 'This block renders on every page that loads its space. Switch to <strong>Only Selected Pages</strong> or <strong>All Pages Except Selected</strong> to limit where it appears.';
          }
        }
      }
      routeCheckboxes().forEach(function (input) {
        input.disabled = !needsSelection;
      });
      updateSelectedCount();
    }

    function setRouteSectionOpen(section, open) {
      if (!section) {
        return;
      }
      var toggle = section.querySelector('.admin-block-route-section-toggle');
      var rows = section.querySelector('.admin-block-route-rows');
      section.classList.toggle('is-open', open);
      if (toggle) {
        toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
      }
      if (rows) {
        rows.hidden = !open;
      }
    }

    function filterRoutes(query) {
      query = (query || '').trim().toLowerCase();
      if (query === '') {
        list.querySelectorAll('.admin-block-route-row, .admin-block-route-item').forEach(function (item) {
          item.classList.remove('is-hidden');
        });
        list.querySelectorAll('.admin-block-route-section, .admin-block-route-group').forEach(function (section) {
          section.classList.remove('is-hidden');
        });
        return;
      }
      list.querySelectorAll('.admin-block-route-row, .admin-block-route-item').forEach(function (item) {
        var label = (item.getAttribute('data-route-label') || item.textContent || '').toLowerCase();
        item.classList.toggle('is-hidden', label.indexOf(query) === -1);
      });
      list.querySelectorAll('.admin-block-route-section, .admin-block-route-group').forEach(function (section) {
        var visible = section.querySelectorAll('.admin-block-route-row:not(.is-hidden), .admin-block-route-item:not(.is-hidden)').length > 0;
        section.classList.toggle('is-hidden', !visible);
        if (visible) {
          setRouteSectionOpen(section, true);
        }
      });
    }

    if (panel.dataset.bound) {
      syncRouteModeUI();
      filterRoutes(searchInput ? searchInput.value : '');
      return;
    }
    panel.dataset.bound = '1';

    modeSelect.addEventListener('change', syncRouteModeUI);
    if (searchInput) {
      searchInput.addEventListener('input', function () {
        filterRoutes(searchInput.value);
      });
    }

    panel.addEventListener('click', function (e) {
      var toggle = e.target.closest('.admin-block-route-section-toggle');
      if (toggle && list.contains(toggle)) {
        var section = toggle.closest('.admin-block-route-section');
        if (section) {
          setRouteSectionOpen(section, !section.classList.contains('is-open'));
        }
        return;
      }
      var selectBtn = e.target.closest('[data-block-route-select]');
      if (selectBtn) {
        var checked = selectBtn.getAttribute('data-block-route-select') === 'all';
        routeCheckboxes().forEach(function (input) {
          if (!input.disabled) {
            input.checked = checked;
          }
        });
        updateSelectedCount();
        return;
      }
    });

    panel.addEventListener('change', function (e) {
      if (e.target && e.target.matches('input[name="route_ids"]')) {
        updateSelectedCount();
      }
    });

    syncRouteModeUI();
    filterRoutes(searchInput ? searchInput.value : '');
  }

  function initMenuItemForm() {
    var menuSelect = document.querySelector('select[name="menu_id"]');
    var parentSelect = document.getElementById('menu-item-parent');
    if (!parentSelect || parentSelect.dataset.bound) {
      return;
    }
    parentSelect.dataset.bound = '1';

    function currentMenuID() {
      if (menuSelect) {
        return menuSelect.value || '';
      }
      return parentSelect.dataset.menuId || '';
    }

    function syncParentOptions() {
      var menuID = currentMenuID();
      parentSelect.querySelectorAll('option').forEach(function (option) {
        if (!option.dataset.menuId) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = menuID && option.dataset.menuId === menuID;
        option.hidden = !matches;
        option.disabled = !matches;
      });
      if (parentSelect.selectedOptions.length && parentSelect.selectedOptions[0].disabled) {
        parentSelect.value = '';
      }
      parentSelect.disabled = !menuID;
    }

    if (menuSelect && !menuSelect.dataset.parentBound) {
      menuSelect.dataset.parentBound = '1';
      menuSelect.addEventListener('change', syncParentOptions);
    }
    syncParentOptions();
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
    var controllerConfigFields = Array.prototype.slice.call(document.querySelectorAll('.route-controller-config-field'));
    var templateOverrideSelect = document.getElementById('route-template-override');
    var templateOverrideField = document.getElementById('route-controller-template-field');
    var helpUrl = document.querySelector('.route-help-url');
    var helpLocal = document.querySelector('.route-help-local');
    var helpIframe = document.querySelector('.route-help-iframe');
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

    function syncControllerTemplateOptions() {
      if (!templateOverrideSelect) {
        return;
      }
      var controllerVisible = typeEl.value === 'Controller' && ctrlEl && !ctrlEl.hidden;
      if (templateOverrideField) {
        templateOverrideField.hidden = !controllerVisible;
      }
      templateOverrideSelect.disabled = !controllerVisible;
      var ctrlID = ctrlSelect ? ctrlSelect.value : '';
      var hasVisible = false;
      templateOverrideSelect.querySelectorAll('option').forEach(function (option) {
        if (!option.dataset.controller) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = controllerVisible && ctrlID && option.dataset.controller === ctrlID;
        option.hidden = !matches;
        option.disabled = !matches;
        if (matches) {
          hasVisible = true;
        }
      });
      if (controllerVisible && ctrlID && templateOverrideSelect.value) {
        var selected = templateOverrideSelect.selectedOptions.length ? templateOverrideSelect.selectedOptions[0] : null;
        if (selected && selected.disabled) {
          templateOverrideSelect.value = '';
        }
      }
      if (templateOverrideField) {
        templateOverrideField.style.display = controllerVisible && hasVisible ? '' : 'none';
      }
    }

    function syncControllerConfigFields() {
      var controllerVisible = typeEl.value === 'Controller' && ctrlEl && !ctrlEl.hidden;
      if (!controllerVisible || !ctrlSelect || !ctrlActionSelect) {
        controllerConfigFields.forEach(function (field) {
          field.hidden = true;
          field.querySelectorAll('input, select, textarea').forEach(function (input) {
            input.disabled = true;
          });
        });
        return;
      }
      var ctrlID = ctrlSelect.value;
      var actionID = ctrlActionSelect.value;
      var feedKind = '';
      if (actionID === 'feed') {
        controllerConfigFields.forEach(function (field) {
          if (field.dataset.controllerId === ctrlID && field.dataset.actionId === actionID && field.dataset.fieldName === 'feed_kind') {
            var select = field.querySelector('select');
            if (select) {
              feedKind = select.value || 'global';
            }
          }
        });
      }
      controllerConfigFields.forEach(function (field) {
        var matches = field.dataset.controllerId === ctrlID && field.dataset.actionId === actionID;
        if (matches && actionID === 'feed' && field.dataset.feedField) {
          matches = field.dataset.feedField === feedKind;
        }
        field.hidden = !matches;
        field.querySelectorAll('input, select, textarea').forEach(function (input) {
          input.disabled = !matches;
        });
      });
      syncControllerTemplateOptions();
    }

    function syncControllerItems() {
      if (!ctrlSelect || !ctrlActionSelect || ctrlEl.hidden) {
        syncControllerConfigFields();
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
      syncControllerConfigFields();
      syncControllerTemplateOptions();
    }

    function syncRouteFields() {
      var type = typeEl.value;
      var showTarget = type === 'Url' || type === 'Local File' || type === 'Iframe';
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
      if (helpIframe) {
        helpIframe.hidden = type !== 'Iframe';
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
        syncControllerConfigFields();
      });
    }
    controllerConfigFields.forEach(function (field) {
      var select = field.querySelector('select[name="controller_data_feed_kind"]');
      if (select) {
        select.addEventListener('change', syncControllerConfigFields);
      }
    });
    syncRouteFields();

    var menuSelect = document.getElementById('route-add-to-menu');
    var menuParentSelect = document.getElementById('route-add-to-menu-parent');
    var routeNameInput = document.querySelector('#route-form input[name="name"]');
    var menuNameInput = document.getElementById('route-add-to-menu-name');

    function syncRouteMenuFields() {
      if (!menuSelect || !menuParentSelect) {
        return;
      }
      var menuID = menuSelect.value || '';
      menuParentSelect.querySelectorAll('option').forEach(function (option) {
        if (!option.dataset.menuId) {
          option.hidden = false;
          option.disabled = false;
          return;
        }
        var matches = menuID && option.dataset.menuId === menuID;
        option.hidden = !matches;
        option.disabled = !matches;
      });
      if (menuParentSelect.selectedOptions.length && menuParentSelect.selectedOptions[0].disabled) {
        menuParentSelect.value = '';
      }
      menuParentSelect.disabled = !menuID;
    }

    if (menuSelect && !menuSelect.dataset.routeMenuBound) {
      menuSelect.dataset.routeMenuBound = '1';
      menuSelect.addEventListener('change', syncRouteMenuFields);
      syncRouteMenuFields();
    }

    if (routeNameInput && menuNameInput && !menuNameInput.dataset.routeMenuBound) {
      menuNameInput.dataset.routeMenuBound = '1';
      routeNameInput.addEventListener('input', function () {
        if (!menuNameInput.value.trim()) {
          menuNameInput.placeholder = routeNameInput.value.trim() || 'Defaults to route name';
        }
      });
    }

    if (window.location.search.indexOf('menu_added=1') !== -1) {
      var menuTab = document.querySelector('.admin-form-sidebar-card [data-sidebar-tab="menu"]');
      if (menuTab) {
        menuTab.click();
      }
    }
  }

  function init() {
    var collapsed = false;
    try {
      collapsed = localStorage.getItem(STORAGE_KEY) === '1';
    } catch (e) {}
    applySidebarCollapsed(collapsed);
    syncNavGroup('nav-content-group');
    syncNavGroup('nav-users-group');
    syncNavGroup('nav-menus-group');
    syncNavGroup('nav-system-group');
    syncNavGroup('nav-extension-apps-group');
    if (collapsed) {
      document.querySelectorAll('.admin-nav-group.open').forEach(positionCollapsedNavFlyout);
    }
    var adminNav = document.querySelector('.admin-nav');
    if (adminNav && !adminNav.dataset.flyoutScrollBound) {
      adminNav.dataset.flyoutScrollBound = '1';
      adminNav.addEventListener('scroll', function () {
        if (!isCollapsed()) {
          return;
        }
        document.querySelectorAll('.admin-nav-group.open').forEach(positionCollapsedNavFlyout);
      }, { passive: true });
    }
    initFormToggles();
    initSidebarTabs();
    initBlockForm();
    initBlockRoutePicker();
    initRouteForm();
    initMenuItemForm();
    initMediaUpload();
    initMediaBrowseUpload();
    initMediaCopy();
    initMediaFolders();
    initItemBulkToolbar();
    initItemForm();
    bindMediaFieldEvents();
    initMediaFields();
    initUserForm();
    initItemEditors();
    initCategoryEditors();
    initTemplateEditors();
    initMediaPicker();
    initTemplateBrowser();
    initAccessLogTail();
    initPermissionPicker();
  }

  function initPermissionPicker() {
    document.querySelectorAll('[data-permission-picker]').forEach(function (picker) {
      if (picker.dataset.bound) {
        return;
      }
      picker.dataset.bound = '1';
      var filter = picker.querySelector('[data-permission-filter]');
      var empty = picker.querySelector('[data-permission-empty]');
      if (!filter) {
        return;
      }
      function applyFilter() {
        var query = filter.value.trim().toLowerCase();
        var visibleRows = 0;
        picker.querySelectorAll('[data-permission-group]').forEach(function (group) {
          var groupVisible = 0;
          group.querySelectorAll('[data-permission-row]').forEach(function (row) {
            var label = (row.getAttribute('data-permission-label') || '').toLowerCase();
            var match = !query || label.indexOf(query) !== -1;
            row.classList.toggle('is-hidden', !match);
            if (match) {
              groupVisible += 1;
              visibleRows += 1;
            }
          });
          group.classList.toggle('is-filter-empty', groupVisible === 0);
        });
        if (empty) {
          empty.hidden = visibleRows !== 0;
        }
      }
      filter.addEventListener('input', applyFilter);
      applyFilter();
      picker.querySelectorAll('[data-permission-allow]').forEach(function (allowBox) {
        allowBox.addEventListener('change', function () {
          if (!allowBox.checked) {
            return;
          }
          var row = allowBox.closest('[data-permission-row]');
          if (!row) {
            return;
          }
          var denyBox = row.querySelector('[data-permission-deny]');
          if (denyBox) {
            denyBox.checked = false;
          }
        });
      });
      picker.querySelectorAll('[data-permission-deny]').forEach(function (denyBox) {
        denyBox.addEventListener('change', function () {
          if (!denyBox.checked) {
            return;
          }
          var row = denyBox.closest('[data-permission-row]');
          if (!row) {
            return;
          }
          var allowBox = row.querySelector('[data-permission-allow]');
          if (allowBox) {
            allowBox.checked = false;
          }
        });
      });
    });
  }

  function initAccessLogTail() {
    var root = document.querySelector('[data-access-log-tail]');
    if (!root || root.dataset.bound) {
      return;
    }
    root.dataset.bound = '1';
    var tailURL = root.getAttribute('data-tail-url') || '';
    var pre = root.querySelector('.admin-access-log-pre');
    var fileSelect = root.querySelector('[data-access-log-file]');
    var refreshBtn = root.querySelector('[data-access-log-refresh]');
    var autoRefresh = root.querySelector('[data-access-log-autorefresh]');
    var timer = null;

    function selectedFile() {
      return fileSelect ? fileSelect.value : 'access.log';
    }

    function refresh() {
      if (!tailURL || !pre) {
        return;
      }
      var url = tailURL + '?file=' + encodeURIComponent(selectedFile());
      fetch(url, { credentials: 'same-origin', cache: 'no-store' })
        .then(function (resp) {
          if (!resp.ok) {
            throw new Error('Failed to load log tail');
          }
          return resp.text();
        })
        .then(function (text) {
          pre.textContent = text || '(empty)';
          pre.parentElement.scrollTop = pre.parentElement.scrollHeight;
        })
        .catch(function () {
          pre.textContent = 'Unable to load access log.';
        });
    }

    function resetTimer() {
      if (timer) {
        clearInterval(timer);
        timer = null;
      }
      if (autoRefresh && autoRefresh.checked) {
        timer = setInterval(refresh, 3000);
      }
    }

    if (fileSelect) {
      fileSelect.addEventListener('change', refresh);
    }
    if (refreshBtn) {
      refreshBtn.addEventListener('click', refresh);
    }
    if (autoRefresh) {
      autoRefresh.addEventListener('change', resetTimer);
    }
    refresh();
    resetTimer();
  }

  function initTemplateBrowser() {
    var browser = document.querySelector('.admin-template-browser');
    if (!browser || browser.dataset.bound) {
      return;
    }
    browser.dataset.bound = '1';

    var expandBtn = document.querySelector('[data-template-browser-expand]');
    var collapseBtn = document.querySelector('[data-template-browser-collapse]');
    if (expandBtn && !expandBtn.dataset.bound) {
      expandBtn.dataset.bound = '1';
      expandBtn.addEventListener('click', function () {
        browser.querySelectorAll('details[data-template-folder]').forEach(function (folder) {
          folder.open = true;
        });
      });
    }
    if (collapseBtn && !collapseBtn.dataset.bound) {
      collapseBtn.dataset.bound = '1';
      collapseBtn.addEventListener('click', function () {
        browser.querySelectorAll('details[data-template-folder]').forEach(function (folder) {
          folder.open = false;
        });
      });
    }
  }

  function loadStylesheet(href) {
    if (document.querySelector('link[href="' + href + '"]')) {
      return Promise.resolve();
    }
    return new Promise(function (resolve, reject) {
      var link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = href;
      link.onload = resolve;
      link.onerror = function () {
        reject(new Error('stylesheet failed to load'));
      };
      document.head.appendChild(link);
    });
  }

  function loadScript(src) {
    if (document.querySelector('script[src="' + src + '"]')) {
      return Promise.resolve();
    }
    return new Promise(function (resolve, reject) {
      var script = document.createElement('script');
      script.src = src;
      script.async = true;
      script.onload = resolve;
      script.onerror = function () {
        reject(new Error('script failed to load'));
      };
      document.head.appendChild(script);
    });
  }

  function loadCodeMirror() {
    if (window.CodeMirror) {
      return Promise.resolve(window.CodeMirror);
    }
    if (codeMirrorLoadPromise) {
      return codeMirrorLoadPromise;
    }
    codeMirrorLoadPromise = loadStylesheet(CODEMIRROR_BASE + '/lib/codemirror.min.css')
      .then(function () {
        return loadScript(CODEMIRROR_BASE + '/lib/codemirror.min.js');
      })
      .then(function () {
        return Promise.all([
          loadScript(CODEMIRROR_BASE + '/mode/xml/xml.min.js'),
          loadScript(CODEMIRROR_BASE + '/mode/javascript/javascript.min.js'),
          loadScript(CODEMIRROR_BASE + '/mode/css/css.min.js'),
          loadScript(CODEMIRROR_BASE + '/mode/markdown/markdown.min.js'),
          loadScript(CODEMIRROR_BASE + '/mode/htmlmixed/htmlmixed.min.js'),
        ]);
      })
      .then(function () {
        if (window.CodeMirror) {
          return window.CodeMirror;
        }
        throw new Error('CodeMirror failed to load');
      });
    return codeMirrorLoadPromise;
  }

  function isTemplateEditorVisible(textarea) {
    if (!textarea || textarea.getAttribute('hidden') !== null) {
      return false;
    }
    var node = textarea;
    while (node) {
      if (node.hidden) {
        return false;
      }
      var style = window.getComputedStyle(node);
      if (style.display === 'none' || style.visibility === 'hidden') {
        return false;
      }
      node = node.parentElement;
    }
    return true;
  }

  function findTemplateEditor(textarea) {
    for (var i = 0; i < templateEditors.length; i++) {
      if (templateEditors[i].textarea === textarea) {
        return templateEditors[i];
      }
    }
    return null;
  }

  function syncTemplateEditors() {
    templateEditors.forEach(function (item) {
      if (item.editor && typeof item.editor.save === 'function') {
        item.editor.save();
      }
    });
  }

  function destroyTemplateEditors() {
    var editors = templateEditors.slice();
    templateEditors = [];
    editors.forEach(function (item) {
      if (item.editor && typeof item.editor.toTextArea === 'function') {
        item.editor.toTextArea();
      }
      if (item.textarea) {
        delete item.textarea.dataset.cmInit;
      }
    });
  }

  function refreshTemplateEditors() {
    templateEditors.forEach(function (item) {
      if (item.editor && typeof item.editor.refresh === 'function') {
        item.editor.refresh();
      }
    });
  }

  function syncBlockContentEditorMode() {
    var typeEl = document.getElementById('block-type');
    var textarea = document.getElementById('block-content');
    if (!typeEl || !textarea) {
      return;
    }
    var mode = typeEl.value === 'markdown' ? 'markdown' : 'htmlmixed';
    textarea.setAttribute('data-editor-mode', mode);
    var item = findTemplateEditor(textarea);
    if (item && item.editor && typeof item.editor.setOption === 'function') {
      item.editor.setOption('mode', mode);
    }
  }

  function bindTemplateEditorSubmit(form) {
    if (!form || form.dataset.templateEditorSubmitBound) {
      return;
    }
    form.dataset.templateEditorSubmitBound = '1';
    form.addEventListener('submit', syncTemplateEditors);
  }

  function initTemplateEditors() {
    var textareas = document.querySelectorAll('textarea.admin-template-editor:not([data-cm-init])');
    if (!textareas.length) {
      return;
    }

    if (!document.body.dataset.templateEditorSubmitBound) {
      document.body.dataset.templateEditorSubmitBound = '1';
      document.addEventListener('turbo:submit-start', function (e) {
        var target = e.target;
        if (target && target.querySelector && target.querySelector('textarea.admin-template-editor')) {
          syncTemplateEditors();
        }
      });
    }

    loadCodeMirror().then(function (CodeMirror) {
      textareas.forEach(function (textarea) {
        if (textarea.dataset.cmInit || !isTemplateEditorVisible(textarea)) {
          return;
        }
        textarea.dataset.cmInit = '1';
        var height = textarea.getAttribute('data-editor-height') || '28rem';
        var editor = CodeMirror.fromTextArea(textarea, {
          mode: textarea.getAttribute('data-editor-mode') || 'htmlmixed',
          lineNumbers: true,
          lineWrapping: true,
          indentUnit: 2,
          tabSize: 2,
          indentWithTabs: false,
          viewportMargin: Infinity,
        });
        editor.setSize(null, height);
        templateEditors.push({ editor: editor, textarea: textarea });
        bindTemplateEditorSubmit(textarea.closest('form'));
      });
      refreshTemplateEditors();
    }).catch(function () {});
  }

  function loadCKEditor5() {
    if (window.ClassicEditor) {
      return Promise.resolve(window.ClassicEditor);
    }
    if (ckEditorLoadPromise) {
      return ckEditorLoadPromise;
    }
    ckEditorLoadPromise = new Promise(function (resolve, reject) {
      var script = document.createElement('script');
      script.src = CKEDITOR5_URL;
      script.async = true;
      script.onload = function () {
        if (window.ClassicEditor) {
          resolve(window.ClassicEditor);
        } else {
          reject(new Error('CKEditor failed to load'));
        }
      };
      script.onerror = function () {
        reject(new Error('CKEditor failed to load'));
      };
      document.head.appendChild(script);
    });
    return ckEditorLoadPromise;
  }

  function syncCKEditors() {
    ckEditorInstances.forEach(function (editor) {
      if (editor && typeof editor.updateSourceElement === 'function') {
        editor.updateSourceElement();
      }
    });
  }

  function destroyCKEditors() {
    var editors = ckEditorInstances.slice();
    ckEditorInstances = [];
    editors.forEach(function (editor) {
      if (editor && typeof editor.destroy === 'function') {
        editor.destroy().catch(function () {});
      }
    });
    ['item-form', 'category-form'].forEach(function (formID) {
      var form = document.getElementById(formID);
      if (form) {
        delete form.dataset.ckEditorInit;
      }
    });
  }

  function initRichTextForm(formID, fields) {
    var form = document.getElementById(formID);
    if (!form || form.dataset.ckEditorInit) {
      return;
    }
    var configs = [];
    fields.forEach(function (field) {
      var el = document.getElementById(field.id);
      if (el && el.getAttribute('data-ckeditor') === field.key) {
        configs.push({ element: el, config: field.config });
      }
    });
    if (!configs.length) {
      return;
    }
    form.dataset.ckEditorInit = '1';
    form.addEventListener('submit', syncCKEditors);

    loadCKEditor5().then(function (ClassicEditor) {
      if (!document.getElementById(formID)) {
        return;
      }
      return Promise.all(configs.map(function (item) {
        return ClassicEditor.create(item.element, item.config);
      })).then(function (editors) {
        if (document.getElementById(formID)) {
          ckEditorInstances = ckEditorInstances.concat(editors);
        } else {
          editors.forEach(function (editor) {
            editor.destroy().catch(function () {});
          });
        }
      });
    }).catch(function (err) {
      console.error('CKEditor failed to initialize:', err);
      delete form.dataset.ckEditorInit;
    });
  }

  var ckEditorIntroToolbar = {
    toolbar: ['bold', 'italic', 'link', '|', 'bulletedList', 'numberedList', '|', 'undo', 'redo']
  };

  var ckEditorBodyToolbar = {
    toolbar: [
      'heading', '|',
      'bold', 'italic', 'link', '|',
      'bulletedList', 'numberedList', 'blockQuote', '|',
      'insertTable', 'mediaEmbed', '|',
      'undo', 'redo'
    ]
  };

  function syncItemEditors() {
    syncCKEditors();
  }

  function destroyItemEditors() {
    destroyCKEditors();
  }

  function initItemEditors() {
    initRichTextForm('item-form', [
      { id: 'item-intro', key: 'intro', config: ckEditorIntroToolbar },
      { id: 'item-body', key: 'body', config: ckEditorBodyToolbar }
    ]);
  }

  function initCategoryEditors() {
    initRichTextForm('category-form', [
      { id: 'category-description', key: 'description', config: ckEditorIntroToolbar }
    ]);
  }

  if (!document.body.dataset.ckEditorSubmitBound) {
    document.body.dataset.ckEditorSubmitBound = '1';
    document.addEventListener('turbo:submit-start', function (e) {
      var target = e.target;
      if (target && (target.id === 'item-form' || target.id === 'category-form' || target.id === 'user-form')) {
        syncCKEditors();
      }
    });
  }

  function initItemMediaRows(form) {
    if (!form || form.dataset.mediaRowsBound) {
      return;
    }
    form.dataset.mediaRowsBound = '1';

    form.querySelectorAll('[data-repeat-add]').forEach(function (button) {
      button.addEventListener('click', function () {
        var kind = button.getAttribute('data-repeat-add');
        var list = form.querySelector('[data-repeat-list="' + kind + '"]');
        if (!list) {
          return;
        }
        var template = list.querySelector('[data-repeat-row="' + kind + '"]');
        if (!template) {
          return;
        }
        var row = template.cloneNode(true);
        row.querySelectorAll('input, select').forEach(function (input) {
          if (input.tagName === 'SELECT') {
            input.selectedIndex = 0;
          } else {
            input.value = '';
          }
        });
        list.appendChild(row);
      });
    });

    form.addEventListener('click', function (event) {
      var removeBtn = event.target.closest('[data-repeat-remove]');
      if (!removeBtn || !form.contains(removeBtn)) {
        return;
      }
      var row = removeBtn.closest('[data-repeat-row]');
      var list = row ? row.parentElement : null;
      if (!row || !list) {
        return;
      }
      if (list.querySelectorAll('[data-repeat-row]').length <= 1) {
        row.querySelectorAll('input, select').forEach(function (input) {
          if (input.tagName === 'SELECT') {
            input.selectedIndex = 0;
          } else {
            input.value = '';
          }
        });
        return;
      }
      row.remove();
    });
  }

  function syncMediaFieldPreview(input) {
    if (!input) {
      return;
    }
    var field = input.closest('[data-media-field]');
    if (!field) {
      return;
    }
    var preview = field.querySelector('[data-media-field-preview]');
    if (!preview) {
      return;
    }
    var url = (input.value || '').trim();
    preview.classList.toggle('is-empty', !url);
    if (!url) {
      preview.innerHTML = '<span class="admin-media-field-placeholder"><i class="bi bi-image"></i> No image selected</span>';
      return;
    }
    preview.innerHTML = '';
    var img = document.createElement('img');
    img.src = url;
    img.alt = input.getAttribute('aria-label') || 'Image preview';
    preview.appendChild(img);
  }

  function initMediaFields(root) {
    var scope = root || document;
    scope.querySelectorAll('[data-media-field] input[type="text"]').forEach(syncMediaFieldPreview);
  }

  function bindMediaFieldEvents() {
    if (document.body.dataset.mediaFieldsBound) {
      return;
    }
    document.body.dataset.mediaFieldsBound = '1';
    document.addEventListener('input', function (e) {
      if (e.target.matches('[data-media-field] input[type="text"]')) {
        syncMediaFieldPreview(e.target);
      }
    });
    document.addEventListener('change', function (e) {
      if (e.target.matches('[data-media-field] input[type="text"]')) {
        syncMediaFieldPreview(e.target);
      }
    });
    document.addEventListener('click', function (e) {
      var clearBtn = e.target.closest('[data-media-picker-clear]');
      if (!clearBtn) {
        return;
      }
      var field = clearBtn.closest('[data-media-field]');
      var input = field && field.querySelector('input[type="text"]');
      if (input) {
        input.value = '';
        syncMediaFieldPreview(input);
      }
    });
  }

  function initItemForm() {
    var form = document.getElementById('item-form');
    if (!form || form.dataset.bound) {
      return;
    }
    form.dataset.bound = '1';

    initSidebarTabs(form);
    initItemMediaRows(form);
    initMediaFields(form);
  }

  function initUserForm() {
    var form = document.getElementById('user-form');
    if (!form || form.dataset.bound) {
      return;
    }
    form.dataset.bound = '1';
    initSidebarTabs(form);
    initLazyCKEditors(form);
    initMediaFields(form);
  }

  function initMediaPicker() {
    if (document.body.dataset.mediaPickerBound) {
      return;
    }
    document.body.dataset.mediaPickerBound = '1';

    var dialog = document.getElementById('media-picker-dialog');
    var frame = document.getElementById('media-picker-frame');
    var imageInput = document.getElementById('item-image');
    var mediaPickerTarget = null;
    var pickerURL = '/admin/media/picker';
    var pickerType = 'images';

    function pickerURLWithParams(folder) {
      var url = pickerURL + '?type=' + encodeURIComponent(pickerType || 'images');
      if (folder) {
        url += '&folder=' + encodeURIComponent(folder);
      }
      return url;
    }

    function openMediaPicker(target, type) {
      mediaPickerTarget = target || imageInput;
      pickerType = type || 'images';
      if (frame) {
        var url = pickerURLWithParams('');
        if (frame.getAttribute('src') === url) {
          frame.reload();
        } else {
          frame.src = url;
        }
      }
      if (dialog && typeof dialog.showModal === 'function') {
        dialog.showModal();
      }
    }

    document.addEventListener('click', function (e) {
      var galleryBtn = e.target.closest('[data-gallery-picker-open]');
      if (galleryBtn) {
        var row = galleryBtn.closest('[data-repeat-row="gallery"]');
        openMediaPicker(row ? row.querySelector('input[name="gallery_url"]') : null, 'images');
        return;
      }
      var attachmentBtn = e.target.closest('[data-attachment-picker-open]');
      if (attachmentBtn) {
        var attachmentRow = attachmentBtn.closest('[data-repeat-row="attachment"]');
        var urlInput = attachmentRow ? attachmentRow.querySelector('input[name="attachment_url"]') : null;
        openMediaPicker(urlInput, 'files');
        return;
      }
      var openTarget = e.target.closest('[data-media-picker-open]');
      if (openTarget) {
        var targetID = openTarget.getAttribute('data-media-picker-target');
        openMediaPicker(targetID ? document.getElementById(targetID) : imageInput, 'images');
        return;
      }
      if (e.target.closest('[data-media-picker-close]')) {
        if (dialog) {
          dialog.close();
        }
        return;
      }
    });

    document.addEventListener('change', function (e) {
      var folderSelect = e.target.closest('[data-media-picker-folder]');
      if (!folderSelect || !frame) {
        return;
      }
      var folder = folderSelect.value || '';
      frame.src = pickerURLWithParams(folder);
    });

    if (dialog) {
      dialog.addEventListener('click', function (e) {
        if (e.target === dialog) {
          dialog.close();
        }
      });
      dialog.addEventListener('click', function (e) {
        var pick = e.target.closest('[data-media-pick]');
        if (!pick) {
          return;
        }
        e.preventDefault();
        var target = mediaPickerTarget || imageInput;
        if (target) {
          target.value = pick.getAttribute('data-media-path') || '';
          syncMediaFieldPreview(target);
          target.dispatchEvent(new Event('change', { bubbles: true }));
          if (pickerType === 'files' && target.name === 'attachment_url') {
            var attachmentRow = target.closest('[data-repeat-row="attachment"]');
            var labelInput = attachmentRow ? attachmentRow.querySelector('input[name="attachment_label"]') : null;
            if (labelInput && !labelInput.value.trim()) {
              var fileName = (pick.getAttribute('title') || pick.querySelector('.admin-media-picker-name')?.textContent || '').trim();
              if (fileName) {
                labelInput.value = fileName;
              }
            }
          }
        }
        mediaPickerTarget = null;
        dialog.close();
      }, true);
      dialog.addEventListener('close', function () {
        mediaPickerTarget = null;
        if (frame) {
          frame.removeAttribute('src');
          frame.innerHTML = '';
        }
      });
    }
  }

  function initItemBulkToolbar() {
    var actionSelect = document.getElementById('item-bulk-action');
    if (!actionSelect || actionSelect.dataset.bound) {
      return;
    }
    actionSelect.dataset.bound = '1';

    var categorySelect = document.getElementById('bulk-category');
    var tagsSelect = document.getElementById('bulk-tags');

    function syncBulkFields() {
      var action = actionSelect.value;
      if (categorySelect) {
        var showCategory = action === 'assign_category';
        categorySelect.hidden = !showCategory;
        categorySelect.disabled = !showCategory;
      }
      if (tagsSelect) {
        var showTags = action === 'assign_tags';
        tagsSelect.hidden = !showTags;
        tagsSelect.disabled = !showTags;
      }
    }

    actionSelect.addEventListener('change', syncBulkFields);
    syncBulkFields();
  }

  function initMediaFolders() {
    var subfolderBtn = document.getElementById('media-add-subfolder-btn');
    var folderSelect = document.getElementById('media-folder');
    var parentSelect = document.getElementById('media-folder-parent');
    if (subfolderBtn && folderSelect && parentSelect && !subfolderBtn.dataset.bound) {
      subfolderBtn.dataset.bound = '1';
      subfolderBtn.addEventListener('click', function () {
        parentSelect.value = folderSelect.value || '';
        var folderPanel = document.querySelector('[data-media-new-folder-panel]');
        if (folderPanel && folderPanel.tagName === 'DETAILS') {
          folderPanel.setAttribute('open', '');
        }
        var nameInput = document.getElementById('media-folder-name');
        if (nameInput) {
          nameInput.focus();
        }
      });
    }

    document.querySelectorAll('[data-media-new-folder-toggle]').forEach(function (btn) {
      if (btn.dataset.bound) {
        return;
      }
      btn.dataset.bound = '1';
      btn.addEventListener('click', function () {
        var panel = document.querySelector('[data-media-new-folder-panel]');
        if (!panel) {
          return;
        }
        panel.classList.toggle('hidden');
        if (!panel.classList.contains('hidden')) {
          var nameInput = panel.querySelector('input[name="name"]');
          if (nameInput) {
            nameInput.focus();
          }
        }
      });
    });

    var folderPanel = document.querySelector('[data-media-new-folder-panel]');
    if (folderPanel) {
      var nameInput = folderPanel.querySelector('input[name="name"]');
      var hasError = folderPanel.querySelector('.admin-alert-danger');
      var shouldOpen = hasError || (nameInput && nameInput.value);
      if (shouldOpen) {
        if (folderPanel.tagName === 'DETAILS') {
          folderPanel.setAttribute('open', '');
        } else {
          folderPanel.classList.remove('hidden');
        }
      }
    }
  }

  function formatFileSize(bytes) {
    if (!bytes || bytes < 1024) {
      return (bytes || 0) + ' B';
    }
    var units = ['KB', 'MB', 'GB'];
    var value = bytes;
    for (var i = 0; i < units.length; i++) {
      value /= 1024;
      if (value < 1024 || i === units.length - 1) {
        return (value >= 100 || units[i] === 'KB' ? value.toFixed(0) : value.toFixed(1)) + ' ' + units[i];
      }
    }
    return bytes + ' B';
  }

  function initMediaBrowseUpload() {
    var form = document.getElementById('media-browse-upload-form');
    if (!form || form.dataset.bound) {
      return;
    }
    var input = form.querySelector('.admin-media-quick-upload-input');
    if (!input) {
      return;
    }
    form.dataset.bound = '1';
    input.addEventListener('change', function () {
      if (input.files && input.files.length) {
        form.submit();
      }
    });
  }

  function initMediaUpload() {
    var form = document.getElementById('media-upload-form');
    var input = document.getElementById('media-file-input');
    var dropzone = document.getElementById('media-dropzone');
    if (!form || !input || !dropzone || dropzone.dataset.bound) {
      return;
    }
    dropzone.dataset.bound = '1';

    var emptyState = document.getElementById('media-dropzone-empty');
    var selectedState = document.getElementById('media-dropzone-selected');
    var selectedName = document.getElementById('media-selected-name');
    var selectedSize = document.getElementById('media-selected-size');
    var selectedIcon = document.getElementById('media-selected-icon');
    var previewBody = document.getElementById('media-preview-body');
    var previewEmpty = document.getElementById('media-preview-empty');
    var previewImage = document.getElementById('media-preview-image');
    var submitBtn = document.getElementById('media-upload-submit');
    var clearBtn = document.getElementById('media-clear-file');
    var previewURL = null;

    function setHidden(el, hidden) {
      if (!el) {
        return;
      }
      el.classList.toggle('hidden', hidden);
    }

    function resetPreviewURL() {
      if (previewURL) {
        URL.revokeObjectURL(previewURL);
        previewURL = null;
      }
    }

    function renderFile(file) {
      if (!file) {
        setHidden(emptyState, false);
        setHidden(selectedState, true);
        setHidden(previewEmpty, false);
        setHidden(previewImage, true);
        if (previewImage) {
          previewImage.removeAttribute('src');
        }
        if (submitBtn) {
          submitBtn.disabled = true;
        }
        resetPreviewURL();
        return;
      }

      setHidden(emptyState, true);
      setHidden(selectedState, false);
      if (selectedName) {
        selectedName.textContent = file.name;
      }
      if (selectedSize) {
        selectedSize.textContent = formatFileSize(file.size);
      }

      var isImage = file.type && file.type.indexOf('image/') === 0;
      if (selectedIcon) {
        selectedIcon.innerHTML = isImage
          ? '<img src="" alt="" id="media-selected-thumb">'
          : '<i class="bi bi-file-earmark"></i>';
      }
      if (isImage && selectedIcon) {
        var thumb = document.getElementById('media-selected-thumb');
        resetPreviewURL();
        previewURL = URL.createObjectURL(file);
        if (thumb) {
          thumb.src = previewURL;
        }
        if (previewImage) {
          previewImage.src = previewURL;
        }
        setHidden(previewEmpty, true);
        setHidden(previewImage, false);
      } else {
        resetPreviewURL();
        setHidden(previewEmpty, false);
        setHidden(previewImage, true);
        if (previewImage) {
          previewImage.removeAttribute('src');
        }
      }

      if (submitBtn) {
        submitBtn.disabled = false;
      }
    }

    function handleFiles(fileList) {
      if (!fileList || !fileList.length) {
        renderFile(null);
        return;
      }
      var file = fileList[0];
      if (window.DataTransfer) {
        var dt = new DataTransfer();
        dt.items.add(file);
        input.files = dt.files;
      }
      renderFile(file);
    }

    dropzone.addEventListener('click', function (e) {
      if (e.target.closest('#media-clear-file')) {
        return;
      }
      input.click();
    });

    input.addEventListener('change', function () {
      handleFiles(input.files);
    });

    if (clearBtn) {
      clearBtn.addEventListener('click', function (e) {
        e.preventDefault();
        e.stopPropagation();
        input.value = '';
        renderFile(null);
        input.click();
      });
    }

    ['dragenter', 'dragover'].forEach(function (eventName) {
      dropzone.addEventListener(eventName, function (e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.add('is-dragover');
      });
    });
    ['dragleave', 'drop'].forEach(function (eventName) {
      dropzone.addEventListener(eventName, function (e) {
        e.preventDefault();
        e.stopPropagation();
        dropzone.classList.remove('is-dragover');
      });
    });
    dropzone.addEventListener('drop', function (e) {
      var files = e.dataTransfer && e.dataTransfer.files;
      handleFiles(files);
    });
  }

  function initMediaCopy() {
    document.querySelectorAll('[data-copy-text]').forEach(function (btn) {
      if (btn.dataset.bound) {
        return;
      }
      btn.dataset.bound = '1';
      btn.addEventListener('click', function () {
        var text = btn.getAttribute('data-copy-text') || '';
        if (!text) {
          return;
        }
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).catch(function () {});
        }
      });
    });
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
    if (isCollapsed() && !e.target.closest('.admin-nav-group')) {
      closeCollapsedNavFlyouts(null);
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
    var open = !group.classList.contains('open');
    if (isCollapsed()) {
      closeCollapsedNavFlyouts(open ? group : null);
    }
    group.classList.toggle('open', open);
    toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (open && isCollapsed()) {
      positionCollapsedNavFlyout(group);
    } else {
      clearNavFlyoutPosition(group);
    }
    try {
      localStorage.setItem(NAV_GROUP_PREFIX + group.id + '-open', open ? '1' : '0');
    } catch (err) {}
  });

  window.addEventListener('resize', function () {
    if (!isCollapsed()) {
      return;
    }
    document.querySelectorAll('.admin-nav-group.open').forEach(positionCollapsedNavFlyout);
  });

  document.addEventListener('turbo:before-cache', function () {
    destroyItemEditors();
    destroyTemplateEditors();
    document.querySelectorAll('[data-bound]').forEach(function (el) {
      delete el.dataset.bound;
    });
    delete document.body.dataset.mediaPickerBound;
  });

  document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('turbo:load', init);
})();
