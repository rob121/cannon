(function () {
  var STORAGE_KEY = 'cannon-admin-sidebar-collapsed';
  var NAV_GROUP_PREFIX = 'cannon-admin-nav-';
  var CKEDITOR5_URL = 'https://cdn.jsdelivr.net/npm/@ckeditor/ckeditor5-build-classic@41.4.2/build/ckeditor.js';
  var CODEMIRROR_BASE = 'https://cdn.jsdelivr.net/npm/codemirror@5.65.16';
  var ckEditorLoadPromise = null;
  var codeMirrorLoadPromise = null;
  var ckEditorInstances = [];
  var ckEditorSourceRecords = [];
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
      if (el.dataset.ckEditorReady && !findCKEditorInstanceForElement(el)) {
        resetCKEditorSourceElement(el);
      }
      if (!el.id || el.dataset.ckEditorReady) {
        return;
      }
      if (el.dataset.ckEditorSourceMode === '1') {
        return;
      }
      var key = el.getAttribute('data-ckeditor') || el.id;
      var config = ckEditorConfig(key === 'body' ? 'body' : 'intro');
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
        return createCKEditor(ClassicEditor, item.element, item.config).then(function (editor) {
          item.element.dataset.ckEditorReady = '1';
          registerCKEditor(editor, item.element, item.config);
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
      var isAdminDash = typeEl.value.indexOf('admin-') === 0;
      var contentEl = document.getElementById('block-fields-content');
      var loginEl = document.getElementById('block-fields-login');
      var menuEl = document.getElementById('block-fields-menu');
      var searchEl = document.getElementById('block-fields-search');
      var adminDashEl = document.getElementById('block-fields-admin-dashboard');
      nativeEl.style.display = (isExtension || isContent || isLogin || isMenu || isSearch || isAdminDash) ? 'none' : '';
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
      if (adminDashEl) {
        adminDashEl.hidden = !isAdminDash;
        adminDashEl.style.display = isAdminDash ? '' : 'none';
      }
      extEl.hidden = !isExtension;
      extEl.style.display = isExtension ? '' : 'none';
      syncBlockItems();

      var contentTextarea = nativeEl.querySelector('textarea.admin-template-editor');
      if (contentTextarea) {
        contentTextarea.setAttribute('data-editor-mode', typeEl.value === 'markdown' ? 'markdown' : 'htmlmixed');
      }
      var showNative = !(isExtension || isContent || isLogin || isMenu || isSearch || isAdminDash);
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

  function initNotificationShoutrr() {
    document.querySelectorAll('[data-notif-shoutrr]').forEach(function (root) {
      if (root.dataset.bound) {
        return;
      }
      root.dataset.bound = '1';
      var source = root.querySelector('[data-notif-shoutrr-source]');
      var custom = root.querySelector('[data-notif-shoutrr-custom]');
      var site = root.querySelector('[data-notif-shoutrr-site]');
      var url = root.querySelector('[data-notif-shoutrr-url]');
      var to = root.querySelector('[data-notif-shoutrr-to]');
      if (!source || !custom || !site) {
        return;
      }
      function sync() {
        var useSite = source.value === 'site_mail';
        custom.hidden = useSite;
        site.hidden = !useSite;
        if (url) {
          url.required = !useSite;
          if (useSite) {
            url.removeAttribute('required');
          }
        }
        if (to) {
          to.required = useSite;
          if (!useSite) {
            to.removeAttribute('required');
          }
        }
      }
      source.addEventListener('change', sync);
      sync();
    });
  }

  function initBlockSpaces() {
    document.querySelectorAll('[data-block-spaces]').forEach(function (root) {
      if (root.dataset.bound) {
        return;
      }
      root.dataset.bound = '1';
      var list = root.querySelector('[data-block-spaces-list]');
      var input = root.querySelector('[data-block-space-input]');
      var addBtn = root.querySelector('[data-block-space-add]');
      var suggest = root.querySelector('[data-block-space-suggest]');
      if (!list || !input || !addBtn) {
        return;
      }

      var knownSpaces = (root.dataset.knownSpaces || '').split(',').map(function (s) {
        return s.trim();
      }).filter(Boolean);

      function existingSpaces() {
        return Array.prototype.map.call(list.querySelectorAll('input[name="spaces"]'), function (el) {
          return (el.value || '').trim();
        }).filter(Boolean);
      }

      function addSpace(value) {
        var space = (value || '').trim();
        if (!space) {
          return;
        }
        if (existingSpaces().indexOf(space) !== -1) {
          input.value = '';
          if (suggest) {
            suggest.value = '';
          }
          return;
        }
        var chip = document.createElement('span');
        chip.className = 'admin-block-space-chip';
        chip.innerHTML = '<input type="hidden" name="spaces" value="">' +
          '<code class="admin-code"></code>' +
          '<button type="button" class="admin-block-space-remove" data-block-space-remove aria-label="Remove space">&times;</button>';
        chip.querySelector('input').value = space;
        chip.querySelector('code').textContent = space;
        list.appendChild(chip);
        input.value = '';
        if (suggest) {
          suggest.value = '';
        }
        input.focus();
      }

      function fillSuggest(names) {
        if (!suggest) {
          return;
        }
        var current = suggest.value;
        suggest.innerHTML = '';
        var placeholder = document.createElement('option');
        placeholder.value = '';
        placeholder.textContent = 'Select a space…';
        suggest.appendChild(placeholder);
        (names || []).forEach(function (name) {
          var opt = document.createElement('option');
          opt.value = name;
          opt.textContent = name;
          suggest.appendChild(opt);
        });
        if (current && names.indexOf(current) !== -1) {
          suggest.value = current;
        }
      }

      function syncSuggestFromExtension() {
        var itemSelect = document.getElementById('block-extension-item');
        var typeEl = document.getElementById('block-type');
        if (!suggest) {
          return;
        }
        if (typeEl && typeEl.value === 'extension' && itemSelect && itemSelect.selectedOptions.length) {
          var selected = itemSelect.selectedOptions[0];
          var spacesAttr = (selected.getAttribute('data-spaces') || '').trim();
          if (spacesAttr) {
            fillSuggest(spacesAttr.split(',').map(function (s) { return s.trim(); }).filter(Boolean));
            return;
          }
        }
        fillSuggest(knownSpaces);
      }

      addBtn.addEventListener('click', function () {
        addSpace(input.value);
      });
      input.addEventListener('keydown', function (e) {
        if (e.key === 'Enter') {
          e.preventDefault();
          addSpace(input.value);
        }
      });
      if (suggest) {
        suggest.addEventListener('change', function () {
          if (suggest.value) {
            addSpace(suggest.value);
          }
        });
      }
      list.addEventListener('click', function (e) {
        var btn = e.target.closest('[data-block-space-remove]');
        if (!btn) {
          return;
        }
        var chip = btn.closest('.admin-block-space-chip');
        if (chip) {
          chip.remove();
        }
      });

      var itemSelect = document.getElementById('block-extension-item');
      var typeEl = document.getElementById('block-type');
      var extSelect = document.getElementById('block-extension');
      if (itemSelect) {
        itemSelect.addEventListener('change', syncSuggestFromExtension);
      }
      if (extSelect) {
        extSelect.addEventListener('change', syncSuggestFromExtension);
      }
      if (typeEl) {
        typeEl.addEventListener('change', syncSuggestFromExtension);
      }
      syncSuggestFromExtension();
    });
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

    function expandSectionsWithSelections() {
      list.querySelectorAll('.admin-block-route-section, .admin-block-route-group').forEach(function (section) {
        var hasChecked = section.querySelector('input[name="route_ids"]:checked');
        if (hasChecked) {
          setRouteSectionOpen(section, true);
        }
      });
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
      if (needsSelection) {
        expandSectionsWithSelections();
      }
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

    var routeForm = document.getElementById('route-form');
    if (routeForm && !routeForm.dataset.ckEditorSubmitBound) {
      routeForm.dataset.ckEditorSubmitBound = '1';
      routeForm.addEventListener('submit', syncCKEditors);
    }

    function syncPageDataFields() {
      syncCKEditors();
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
      initVisibleRouteEditors();
    }

    function initVisibleRouteEditors() {
      document.querySelectorAll('.route-data-field:not([hidden])').forEach(function (field) {
        initLazyCKEditors(field);
      });
      refreshCKEditors();
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

  function initTurboProgress() {
    if (document.body.dataset.turboProgressBound) {
      return;
    }
    document.body.dataset.turboProgressBound = '1';
    var bar = document.getElementById('admin-turbo-progress');
    if (!bar) {
      return;
    }
    var timer = null;

    function start() {
      if (timer) {
        clearTimeout(timer);
        timer = null;
      }
      bar.classList.add('turbo-progress-bar--visible');
      bar.style.width = '20%';
      window.requestAnimationFrame(function () {
        bar.style.width = '60%';
      });
    }

    function finish() {
      bar.style.width = '100%';
      timer = setTimeout(function () {
        bar.classList.remove('turbo-progress-bar--visible');
        bar.style.width = '0';
      }, 180);
    }

    document.addEventListener('turbo:visit', start);
    document.addEventListener('turbo:load', finish);
    document.addEventListener('turbo:frame-load', finish);
  }

  // Keep ?tp=1 across admin Turbo navigations so space outlines stay enabled.
  function initPreserveTemplateSpaceDebug() {
    if (document.body.dataset.tpPreserveBound) {
      return;
    }
    document.body.dataset.tpPreserveBound = '1';

    function wantsTP() {
      return document.body.dataset.debugTemplateSpaces === '1' ||
        new URLSearchParams(window.location.search).get('tp') === '1';
    }

    function withTP(href) {
      try {
        var url = new URL(href, window.location.origin);
        if (url.origin !== window.location.origin) {
          return href;
        }
        if (!url.pathname.startsWith('/admin')) {
          return href;
        }
        if (url.searchParams.get('tp') === '1') {
          return url.pathname + url.search + url.hash;
        }
        url.searchParams.set('tp', '1');
        return url.pathname + url.search + url.hash;
      } catch (e) {
        return href;
      }
    }

    document.addEventListener('click', function (event) {
      if (!wantsTP()) {
        return;
      }
      var link = event.target && event.target.closest ? event.target.closest('a[href]') : null;
      if (!link || link.target === '_blank' || link.hasAttribute('download')) {
        return;
      }
      var raw = link.getAttribute('href');
      if (!raw || raw.charAt(0) === '#') {
        return;
      }
      var next = withTP(raw);
      if (next !== raw) {
        link.setAttribute('href', next);
      }
    }, true);

    document.addEventListener('turbo:before-fetch-request', function (event) {
      if (!wantsTP()) {
        return;
      }
      try {
        var url = new URL(event.detail.url.toString(), window.location.origin);
        if (!url.pathname.startsWith('/admin') || url.searchParams.get('tp') === '1') {
          return;
        }
        url.searchParams.set('tp', '1');
        event.detail.url = url;
      } catch (e) {}
    });
  }

  function initUserMenu() {
    if (document.body.dataset.userMenuBound) {
      return;
    }
    document.body.dataset.userMenuBound = '1';
    var menu = document.getElementById('admin-user-menu');
    var toggle = document.getElementById('admin-user-menu-toggle');
    var dropdown = document.getElementById('admin-user-menu-dropdown');
    if (!menu || !toggle || !dropdown) {
      return;
    }

    function closeMenu() {
      menu.classList.remove('open');
      toggle.setAttribute('aria-expanded', 'false');
      dropdown.hidden = true;
    }

    function openMenu() {
      menu.classList.add('open');
      toggle.setAttribute('aria-expanded', 'true');
      dropdown.hidden = false;
    }

    toggle.addEventListener('click', function (e) {
      e.stopPropagation();
      if (menu.classList.contains('open')) {
        closeMenu();
      } else {
        openMenu();
      }
    });

    document.addEventListener('click', function (e) {
      if (!menu.contains(e.target)) {
        closeMenu();
      }
    });

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') {
        closeMenu();
      }
    });
  }

  function collectCommandItems() {
    var items = [];
    var seen = {};
    document.querySelectorAll('.admin-nav a.admin-nav-link, .admin-nav a.admin-nav-sublink').forEach(function (link) {
      var href = link.getAttribute('href');
      if (!href || href === '#') {
        return;
      }
      var label = '';
      var span = link.querySelector('span');
      if (span) {
        label = span.textContent.trim();
      }
      if (!label) {
        label = link.textContent.trim();
      }
      if (!label || seen[href + '|' + label]) {
        return;
      }
      seen[href + '|' + label] = true;
      var iconEl = link.querySelector('i.bi');
      var icon = iconEl ? iconEl.className : 'bi bi-link-45deg';
      var path = '';
      var group = link.closest('.admin-nav-group');
      if (group) {
        var groupLabel = group.querySelector('.admin-nav-group-toggle span');
        if (groupLabel) {
          path = groupLabel.textContent.trim();
        }
      }
      items.push({
        label: label,
        href: href,
        icon: icon,
        path: path,
        target: link.getAttribute('target') || '',
        searchText: (path + ' ' + label).toLowerCase()
      });
    });
    var siteLink = document.querySelector('.admin-topbar-site-link');
    if (siteLink && siteLink.href) {
      items.push({
        label: 'View Site',
        href: siteLink.href,
        icon: 'bi bi-box-arrow-up-right',
        path: 'Site',
        target: '_blank',
        searchText: 'view site open frontend'
      });
    }
    return items;
  }

  function initCommandPalette() {
    if (document.body.dataset.commandPaletteBound) {
      return;
    }
    document.body.dataset.commandPaletteBound = '1';

    var openBtn = document.getElementById('admin-command-open');
    var dialog = document.getElementById('admin-command-dialog');
    var input = document.getElementById('admin-command-input');
    var results = document.getElementById('admin-command-results');
    if (!openBtn || !dialog || !input || !results) {
      return;
    }

    var items = collectCommandItems();
    var activeIndex = 0;
    var filtered = items.slice();

    var isMac = /Mac|iPhone|iPad|iPod/.test(navigator.platform) || /Mac OS X/.test(navigator.userAgent);
    document.querySelectorAll('.admin-topbar-kbd').forEach(function (kbd) {
      kbd.textContent = isMac ? '⌘K' : 'Ctrl+K';
    });

    function renderResults() {
      results.innerHTML = '';
      if (!filtered.length) {
        var empty = document.createElement('div');
        empty.className = 'admin-command-empty';
        empty.textContent = 'No matching pages';
        results.appendChild(empty);
        return;
      }
      filtered.forEach(function (item, index) {
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'admin-command-item' + (index === activeIndex ? ' active' : '');
        btn.setAttribute('role', 'option');
        btn.setAttribute('aria-selected', index === activeIndex ? 'true' : 'false');
        btn.innerHTML = '<i class="' + item.icon + '" aria-hidden="true"></i><span>' + item.label + '</span>' +
          (item.path ? '<span class="admin-command-item-path">' + item.path + '</span>' : '');
        btn.addEventListener('click', function () {
          navigateTo(item);
        });
        results.appendChild(btn);
      });
    }

    function filterItems(query) {
      query = (query || '').trim().toLowerCase();
      if (!query) {
        filtered = items.slice();
      } else {
        filtered = items.filter(function (item) {
          return item.searchText.indexOf(query) !== -1 || item.label.toLowerCase().indexOf(query) !== -1;
        });
      }
      activeIndex = 0;
      renderResults();
    }

    function navigateTo(item) {
      closePalette();
      if (!item || !item.href) {
        return;
      }
      if (item.target === '_blank') {
        window.open(item.href, '_blank', 'noopener');
        return;
      }
      if (window.Turbo && typeof window.Turbo.visit === 'function') {
        window.Turbo.visit(item.href, { frame: '_top' });
      } else {
        window.location.href = item.href;
      }
    }

    function openPalette() {
      items = collectCommandItems();
      input.value = '';
      filterItems('');
      if (typeof dialog.showModal === 'function') {
        dialog.showModal();
      }
      window.requestAnimationFrame(function () {
        input.focus();
        input.select();
      });
    }

    function closePalette() {
      if (dialog.open) {
        dialog.close();
      }
    }

    openBtn.addEventListener('click', openPalette);

    dialog.addEventListener('click', function (e) {
      if (e.target === dialog) {
        closePalette();
      }
    });

    dialog.addEventListener('close', function () {
      input.value = '';
      activeIndex = 0;
    });

    input.addEventListener('input', function () {
      filterItems(input.value);
    });

    input.addEventListener('keydown', function (e) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        if (!filtered.length) {
          return;
        }
        activeIndex = (activeIndex + 1) % filtered.length;
        renderResults();
        var active = results.querySelector('.admin-command-item.active');
        if (active) {
          active.scrollIntoView({ block: 'nearest' });
        }
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        if (!filtered.length) {
          return;
        }
        activeIndex = (activeIndex - 1 + filtered.length) % filtered.length;
        renderResults();
        var activeUp = results.querySelector('.admin-command-item.active');
        if (activeUp) {
          activeUp.scrollIntoView({ block: 'nearest' });
        }
        return;
      }
      if (e.key === 'Enter') {
        e.preventDefault();
        if (filtered[activeIndex]) {
          navigateTo(filtered[activeIndex]);
        }
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        closePalette();
      }
    });

    document.addEventListener('keydown', function (e) {
      var mod = isMac ? e.metaKey : e.ctrlKey;
      if (mod && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault();
        if (dialog.open) {
          closePalette();
        } else {
          openPalette();
        }
      }
    });
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
    initBlockSpaces();
    initNotificationShoutrr();
    initBlockRoutePicker();
    initRouteForm();
    initMenuItemForm();
    initMediaUpload();
    initExtensionUpload();
    initMediaBrowseUpload();
    initMediaCopy();
    initMediaFolders();
    initItemBulkToolbar();
    initItemMediaRows();
    initItemForm();
    bindMediaFieldEvents();
    initMediaFields();
    initUserForm();
    initItemEditors();
    initCategoryEditors();
    initExtensionRichText();
    initTemplateEditors();
    initMediaPicker();
    initTemplateBrowser();
    initAccessLogTail();
    initSystemUpdate();
    initRealtimeAnalytics();
    initPermissionPicker();
    initTurboProgress();
    initPreserveTemplateSpaceDebug();
    initUserMenu();
    initCommandPalette();
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

  function initSystemUpdate() {
    var root = document.querySelector('[data-system-update]');
    if (!root || root.dataset.bound) {
      return;
    }
    root.dataset.bound = '1';

    var statusURL = root.getAttribute('data-status-url') || '';
    var actionURL = root.getAttribute('data-action-url') || '';
    var progress = root.querySelector('[data-system-update-progress]');
    var msgEl = root.querySelector('[data-system-update-msg]');
    var flash = root.querySelector('[data-system-update-flash]');
    var busy = false;
    var pollTimer = null;

    function setText(sel, value) {
      var el = root.querySelector(sel);
      if (el) {
        el.textContent = value == null ? '' : String(value);
      }
    }

    function setHidden(sel, hidden) {
      root.querySelectorAll(sel).forEach(function (el) {
        el.hidden = !!hidden;
      });
    }

    function setBusy(next) {
      busy = !!next;
      root.querySelectorAll('[data-system-update-btn]').forEach(function (btn) {
        btn.disabled = busy;
      });
    }

    function showFlash(kind, text) {
      if (!flash) {
        return;
      }
      if (!text) {
        flash.innerHTML = '';
        return;
      }
      var isError = kind === 'error';
      var cls = isError ? 'admin-alert-error' : 'admin-alert-success';
      var icon = isError ? 'bi-exclamation-triangle' : 'bi-check-circle';
      flash.innerHTML =
        '<div class="admin-alert ' + cls + '" role="' + (isError ? 'alert' : 'status') + '">' +
          '<i class="bi ' + icon + '" aria-hidden="true"></i> ' +
          escapeHTML(text) +
        '</div>';
    }

    function escapeHTML(value) {
      return String(value || '')
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
    }

    function formatCheckedAt(iso) {
      if (!iso) {
        return '';
      }
      var d = new Date(iso);
      if (isNaN(d.getTime())) {
        return String(iso);
      }
      return d.toLocaleString(undefined, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: 'numeric',
        minute: '2-digit'
      });
    }

    function setProgress(activeStep, message, mode) {
      if (!progress) {
        return;
      }
      var show = !!activeStep || !!message;
      progress.hidden = !show;
      if (mode) {
        progress.setAttribute('data-mode', mode);
      }
      progress.querySelectorAll('[data-step]').forEach(function (li) {
        var step = li.getAttribute('data-step');
        li.classList.remove('is-active', 'is-done', 'is-error');
        if (!activeStep) {
          return;
        }
        var order = ['check', 'download', 'restart', 'verify'];
        var activeIdx = order.indexOf(activeStep);
        var idx = order.indexOf(step);
        if (idx < 0) {
          return;
        }
        if (idx < activeIdx) {
          li.classList.add('is-done');
        } else if (idx === activeIdx) {
          li.classList.add(mode === 'error' ? 'is-error' : 'is-active');
        }
      });
      if (msgEl) {
        msgEl.textContent = message || '';
      }
    }

    function clearProgressSoon() {
      window.setTimeout(function () {
        if (!busy) {
          setProgress('', '', '');
          if (progress) {
            progress.hidden = true;
          }
        }
      }, 4000);
    }

    function applyStatus(status) {
      if (!status) {
        return;
      }
      root.setAttribute('data-current-version', status.current_version || '');
      setText('[data-field="current_version"]', status.current_version || '');
      setText('[data-field="commit"]', status.commit || '');
      setHidden('[data-commit-row]', !status.commit);
      setText('[data-field="binary_path"]', status.binary_path || '');
      var urlEl = root.querySelector('[data-field="update_url_base"]');
      if (urlEl) {
        urlEl.textContent = status.update_url_base || '';
        urlEl.setAttribute('href', status.update_url_base || '#');
      }

      var latest = status.latest_version || '';
      var available = !!(status.update_available && !status.has_staged);
      var upToDate = !!(latest && !status.update_available && !status.has_staged);
      setHidden('[data-latest-row]', !latest);
      setText('[data-field="latest_version"]', latest);

      var stateBadge = root.querySelector('[data-field="status_badge"]');
      if (stateBadge) {
        stateBadge.classList.remove('is-current', 'is-available', 'is-staged');
        if (status.has_staged) {
          stateBadge.classList.add('is-staged');
          stateBadge.textContent = 'Staged';
        } else if (available) {
          stateBadge.classList.add('is-available');
          stateBadge.textContent = 'Update available';
        } else if (upToDate) {
          stateBadge.classList.add('is-current');
          stateBadge.textContent = 'Up to date';
        } else {
          stateBadge.textContent = 'Not checked';
        }
      }

      var badge = root.querySelector('[data-field="latest_badge"]');
      if (badge) {
        badge.classList.toggle('admin-badge-warning', !!status.update_available);
        badge.classList.toggle('admin-badge-success', !status.update_available);
        badge.textContent = status.update_available ? 'New' : 'Current';
      }
      setText('[data-field="apply_version"]', latest);

      var stagedTile = root.querySelector('[data-file-staged]');
      if (stagedTile) {
        stagedTile.classList.toggle('is-present', !!status.has_staged);
        stagedTile.classList.toggle('is-empty', !status.has_staged);
      }
      var stagedPill = root.querySelector('[data-field="staged_pill"]');
      if (stagedPill) {
        stagedPill.textContent = status.has_staged ? 'Ready' : 'None';
      }
      var stagedEl = root.querySelector('[data-field="staged_file"]');
      if (stagedEl) {
        if (status.has_staged) {
          stagedEl.innerHTML = '<code class="admin-code">' + escapeHTML(status.latest_path) + '</code>' +
            (status.staged_version ? ' <span class="admin-cell-muted">(' + escapeHTML(status.staged_version) + ')</span>' : '');
        } else {
          stagedEl.innerHTML = '<span class="admin-capability-none">No staged update</span>';
        }
      }

      var prevTile = root.querySelector('[data-file-previous]');
      if (prevTile) {
        prevTile.classList.toggle('is-present', !!status.has_previous);
        prevTile.classList.toggle('is-empty', !status.has_previous);
      }
      var prevPill = root.querySelector('[data-field="previous_pill"]');
      if (prevPill) {
        prevPill.textContent = status.has_previous ? 'Available' : 'None';
      }
      var prevEl = root.querySelector('[data-field="previous_file"]');
      if (prevEl) {
        if (status.has_previous) {
          prevEl.innerHTML = '<code class="admin-code">' + escapeHTML(status.previous_path) + '</code>';
        } else {
          prevEl.innerHTML = '<span class="admin-capability-none">No rollback binary</span>';
        }
      }

      setHidden('[data-system-update-staged]', !status.has_staged);
      setText('[data-field="staged_version"]', status.staged_version || '');
      setText('[data-field="latest_path"]', status.latest_path || '');
      setHidden('[data-system-update-restart-hint]', !status.restart_required);
      setHidden('[data-system-update-restart-only]', !(!status.has_staged && status.restart_required));

      var checked = formatCheckedAt(status.update_checked_at);
      setHidden('[data-checked-row]', !checked);
      setText('[data-field="update_checked_at"]', checked);

      setHidden('[data-error-row]', !status.update_error);
      setText('[data-field="update_error"]', status.update_error || '');

      var applyForm = root.querySelector('[data-action-kind="apply"]');
      if (applyForm) {
        applyForm.hidden = !available;
        applyForm.setAttribute(
          'data-confirm',
          'Download Cannon ' + latest + ' to cannon.latest and restart so it can be applied?'
        );
      }
      var discardForm = root.querySelector('[data-action-kind="discard"]');
      if (discardForm) {
        discardForm.hidden = !status.has_staged;
      }
      var rollbackForm = root.querySelector('[data-action-kind="rollback"]');
      if (rollbackForm) {
        rollbackForm.hidden = !status.has_previous;
      }
    }

    function fetchStatus() {
      if (!statusURL) {
        return Promise.reject(new Error('status URL missing'));
      }
      return fetch(statusURL, {
        credentials: 'same-origin',
        cache: 'no-store',
        headers: { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' }
      }).then(function (resp) {
        if (!resp.ok) {
          throw new Error('status ' + resp.status);
        }
        return resp.json();
      });
    }

    function stopPolling() {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    }

    function normVer(value) {
      return String(value || '').trim().replace(/^v/i, '');
    }

    function pollUntilReady(baselineVersion, expectedVersion) {
      setProgress('verify', 'Waiting for Cannon to come back…', 'update');
      var attempts = 0;
      var maxAttempts = 60;
      var base = normVer(baselineVersion);
      var expect = normVer(expectedVersion);
      stopPolling();
      return new Promise(function (resolve, reject) {
        function tick() {
          attempts += 1;
          fetchStatus()
            .then(function (status) {
              applyStatus(status);
              var current = normVer(status.current_version);
              var ready = !status.has_staged && !status.restart_required;
              var versionOk = !expect || current === expect || (base && current && current !== base);
              if (ready && versionOk) {
                stopPolling();
                resolve(status);
                return;
              }
              setProgress(
                'verify',
                'Process is back; finishing apply… (attempt ' + attempts + ')',
                'update'
              );
              if (attempts >= maxAttempts) {
                stopPolling();
                reject(new Error('Timed out waiting for update to finish'));
              }
            })
            .catch(function () {
              setProgress(
                'restart',
                'Restarting… waiting for admin to respond (attempt ' + attempts + ')',
                'update'
              );
              if (attempts >= maxAttempts) {
                stopPolling();
                reject(new Error('Timed out waiting for Cannon to restart'));
              }
            });
        }
        pollTimer = setInterval(tick, 1500);
        tick();
      });
    }

    function isRecoverableRestartError(err) {
      if (!err) {
        return false;
      }
      if (err.noPoll) {
        return false;
      }
      var msg = String(err.message || err);
      return err.name === 'TypeError' ||
        err.name === 'SyntaxError' ||
        /Failed to fetch|NetworkError|Load failed|Unexpected end|JSON/i.test(msg);
    }

    function runAction(form) {
      if (busy || !form) {
        return;
      }
      var kind = form.getAttribute('data-action-kind') || '';
      var confirmMsg = form.getAttribute('data-confirm');
      if (confirmMsg && !window.confirm(confirmMsg)) {
        return;
      }

      var baseline = root.getAttribute('data-current-version') || '';
      var expected = '';
      var applyVersionEl = root.querySelector('[data-field="apply_version"]');
      if (kind === 'apply' && applyVersionEl) {
        expected = applyVersionEl.textContent.trim();
      }

      setBusy(true);
      showFlash('', '');
      if (kind === 'check') {
        setProgress('check', 'Checking for updates…', 'check');
      } else if (kind === 'apply') {
        setProgress('download', 'Downloading update…', 'update');
      } else if (kind === 'rollback') {
        setProgress('restart', 'Preparing rollback…', 'update');
      } else if (kind === 'discard') {
        setProgress('check', 'Discarding staged update…', 'check');
      }

      fetch(actionURL || form.action, {
        method: 'POST',
        body: new FormData(form),
        credentials: 'same-origin',
        cache: 'no-store',
        headers: {
          Accept: 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        }
      })
        .then(function (resp) {
          return resp.json().then(function (body) {
            return { ok: resp.ok, body: body };
          });
        })
        .then(function (result) {
          var body = result.body || {};
          if (body.status) {
            applyStatus(body.status);
          }
          if (!result.ok || body.ok === false) {
            throw new Error(body.error || 'Update action failed');
          }
          if (body.restarting) {
            if (kind === 'apply') {
              setProgress('restart', 'Download complete. Restarting to apply ' + (expected || 'update') + '…', 'update');
            } else {
              setProgress('restart', 'Restarting…', 'update');
            }
            return pollUntilReady(baseline, kind === 'apply' ? expected : '').then(function (status) {
              showFlash('success', kind === 'rollback'
                ? ('Rollback complete. Now running ' + (status.current_version || '') + '.')
                : ('Update complete. Now running ' + (status.current_version || '') + '.'));
              setProgress('verify', 'Done.', 'update');
              if (progress) {
                progress.querySelectorAll('[data-step]').forEach(function (li) {
                  li.classList.remove('is-active', 'is-error');
                  li.classList.add('is-done');
                });
              }
              clearProgressSoon();
            }).catch(function (pollErr) {
              var e = pollErr instanceof Error ? pollErr : new Error(String(pollErr));
              e.noPoll = true;
              throw e;
            });
          }
          showFlash('success', body.message || 'Done.');
          if (kind === 'check') {
            setProgress('check', body.status && body.status.update_available
              ? ('Update available: ' + (body.status.latest_version || ''))
              : 'Already up to date.', 'check');
            var checkStep = progress ? progress.querySelector('[data-step="check"]') : null;
            if (checkStep) {
              checkStep.classList.add('is-done');
              checkStep.classList.remove('is-active');
            }
          } else {
            setProgress('', '', '');
          }
          clearProgressSoon();
        })
        .catch(function (err) {
          // Apply/rollback may kill the process before the JSON response finishes.
          if ((kind === 'apply' || kind === 'rollback') && isRecoverableRestartError(err)) {
            setProgress('restart', 'Connection dropped during restart; polling for recovery…', 'update');
            return pollUntilReady(baseline, kind === 'apply' ? expected : '')
              .then(function (status) {
                showFlash('success', 'Now running ' + (status.current_version || '') + '.');
                setProgress('verify', 'Done.', 'update');
                if (progress) {
                  progress.querySelectorAll('[data-step]').forEach(function (li) {
                    li.classList.remove('is-active', 'is-error');
                    li.classList.add('is-done');
                  });
                }
                clearProgressSoon();
              })
              .catch(function (pollErr) {
                showFlash('error', pollErr.message || String(pollErr));
                setProgress('restart', pollErr.message || 'Restart failed', 'error');
              });
          }
          showFlash('error', err.message || String(err));
          setProgress(kind === 'check' ? 'check' : 'download', err.message || 'Failed', 'error');
        })
        .finally(function () {
          setBusy(false);
        });
    }

    root.querySelectorAll('[data-system-update-form]').forEach(function (form) {
      form.addEventListener('submit', function (e) {
        e.preventDefault();
        e.stopPropagation();
        runAction(form);
      });
    });
  }

  function loadExternalScript(src) {
    return new Promise(function (resolve, reject) {
      if (document.querySelector('script[data-src="' + src + '"]')) {
        resolve();
        return;
      }
      var script = document.createElement('script');
      script.src = src;
      script.async = true;
      script.dataset.src = src;
      script.onload = function () { resolve(); };
      script.onerror = function () { reject(new Error('script load failed')); };
      document.head.appendChild(script);
    });
  }

  function parseAnalyticsStats(raw) {
    if (!raw) {
      return null;
    }
    if (typeof raw === 'object' && !(raw instanceof ArrayBuffer) && !(raw instanceof Uint8Array)) {
      return raw;
    }
    var text = raw;
    if (raw instanceof Uint8Array) {
      text = new TextDecoder().decode(raw);
    }
    if (typeof text !== 'string') {
      return null;
    }
    try {
      return JSON.parse(text);
    } catch (err) {
      return null;
    }
  }

  function initRealtimeAnalytics() {
    var panel = document.getElementById('analytics-realtime-panel');
    if (!panel || panel.dataset.bound) {
      return;
    }
    var configEl = document.getElementById('cannon-admin-realtime-config');
    if (!configEl) {
      return;
    }
    var config;
    try {
      config = JSON.parse(configEl.textContent || '{}');
    } catch (err) {
      return;
    }
    if (!config.endpoint || !config.analytics) {
      return;
    }
    panel.dataset.bound = '1';

    function setText(id, value) {
      var el = document.getElementById(id);
      if (el) {
        el.textContent = String(value);
      }
    }

    function renderStats(stats) {
      stats = parseAnalyticsStats(stats);
      if (!stats) {
        return;
      }
      setText('analytics-online-count', stats.online || 0);
      setText('analytics-panel-online', stats.online || 0);
      setText('analytics-panel-auth', stats.authenticated || 0);
      var pages = Array.isArray(stats.pages) ? stats.pages : [];
      setText('analytics-panel-pages', pages.length);
      var body = document.getElementById('analytics-pages-body');
      if (!body) {
        return;
      }
      if (!pages.length) {
        body.innerHTML = '<tr><td colspan="2" class="text-muted">No active visitors</td></tr>';
        return;
      }
      body.innerHTML = pages.map(function (row) {
        var path = row.path || '/';
        var count = row.count || 0;
        return '<tr><td><code class="admin-code">' + path.replace(/</g, '&lt;') + '</code></td><td class="text-end">' + count + '</td></tr>';
      }).join('');
    }

    function fetchLiveStats() {
      return fetch('/admin/analytics/live', { credentials: 'same-origin', cache: 'no-store' })
        .then(function (resp) {
          if (!resp.ok) {
            throw new Error('analytics fetch failed');
          }
          return resp.json();
        })
        .then(renderStats)
        .catch(function () {});
    }

    fetchLiveStats();

    loadExternalScript('https://cdn.jsdelivr.net/npm/centrifuge@5.2.2/dist/centrifuge.js')
      .then(function () {
        if (typeof Centrifuge === 'undefined') {
          return;
        }
        var centrifuge = new Centrifuge(config.endpoint, {});
        var sub = centrifuge.newSubscription(config.analytics);
        sub.on('publication', function (ctx) {
          renderStats(ctx.data);
        });
        sub.on('subscribed', fetchLiveStats);
        centrifuge.connect();
        sub.subscribe();
        window.setInterval(fetchLiveStats, 15000);
      })
      .catch(function () {
        window.setInterval(fetchLiveStats, 5000);
      });
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
    var editorCtor = getCKEditor5Constructor();
    if (editorCtor) {
      return Promise.resolve(editorCtor);
    }
    if (ckEditorLoadPromise) {
      return ckEditorLoadPromise;
    }
    ckEditorLoadPromise = new Promise(function (resolve, reject) {
      var script = document.createElement('script');
      script.src = CKEDITOR5_URL;
      script.async = true;
      script.onload = function () {
        var editorCtor = getCKEditor5Constructor();
        if (editorCtor) {
          resolve(editorCtor);
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

  function getCKEditor5Constructor() {
    if (window.ClassicEditor) {
      return window.ClassicEditor;
    }
    if (window.CKEDITOR && window.CKEDITOR.ClassicEditor) {
      return window.CKEDITOR.ClassicEditor;
    }
    return null;
  }

  function createCKEditor(ClassicEditor, element, config) {
    return ClassicEditor.create(element, config);
  }

  function ckEditorPlugins(kind) {
    var c = window.CKEDITOR;
    if (!c || !c.GeneralHtmlSupport) {
      return null;
    }
    var plugins = [
      c.Essentials,
      c.Paragraph,
      c.GeneralHtmlSupport,
      c.Bold,
      c.Italic,
      c.Link,
      c.List
    ];
    if (kind === 'body') {
      plugins = plugins.concat([
        c.Heading,
        c.BlockQuote,
        c.Table,
        c.TableToolbar,
        c.MediaEmbed
      ]);
    }
    return plugins.filter(Boolean);
  }

  function ckEditorConfig(kind) {
    var toolbar = kind === 'body' ? [
      'heading', '|',
      'bold', 'italic', 'link', '|',
      'bulletedList', 'numberedList', 'blockQuote', '|',
      'insertTable', 'mediaEmbed', '|',
      'undo', 'redo'
    ] : ['bold', 'italic', 'link', '|', 'bulletedList', 'numberedList', '|', 'undo', 'redo'];

    var config = {
      toolbar: toolbar
    };
    var plugins = ckEditorPlugins(kind);
    if (!plugins) {
      return config;
    }

    config.licenseKey = 'GPL';
    config.plugins = plugins;
    config.htmlSupport = {
      allow: [
        {
          name: /^(?!script$).+/,
          styles: true,
          classes: true,
          attributes: [
            'id',
            'title',
            'lang',
            'dir',
            'role',
            'aria-label',
            'aria-describedby',
            /^data-[\w-]+$/
          ]
        }
      ],
      disallow: [
        {
          name: /^(script|iframe|object|embed)$/i
        },
        {
          attributes: /^on.*$/i
        }
      ]
    };
    if (kind === 'body') {
      config.table = {
        contentToolbar: ['tableColumn', 'tableRow', 'mergeTableCells']
      };
    }
    return config;
  }

  function syncCKEditors() {
    ckEditorInstances.forEach(function (editor) {
      if (editor && typeof editor.updateSourceElement === 'function') {
        editor.updateSourceElement();
      }
    });
  }

  function findCKEditorInstanceForElement(element) {
    for (var i = 0; i < ckEditorInstances.length; i += 1) {
      if (ckEditorInstances[i] && ckEditorInstances[i].sourceElement === element) {
        return ckEditorInstances[i];
      }
    }
    return null;
  }

  function resetCKEditorSourceElement(element) {
    if (!element) {
      return;
    }
    delete element.dataset.ckEditorReady;
    delete element.dataset.ckEditorSourceMode;
    element.classList.remove('ck-hidden', 'admin-richtext-source-mode');
    element.hidden = false;
    element.style.display = '';

    var field = element.closest('.admin-richtext-field');
    if (!field) {
      return;
    }
    field.querySelectorAll('.admin-richtext-source-bar').forEach(function (bar) {
      bar.remove();
    });
    field.querySelectorAll('.ck.ck-editor').forEach(function (wrapper) {
      wrapper.remove();
    });
  }

  function resetStaleCKEditorUI(root) {
    var scope = root || document;
    scope.querySelectorAll('textarea[data-ckeditor]').forEach(resetCKEditorSourceElement);
  }

  function destroyCKEditors() {
    var editors = ckEditorInstances.slice();
    ckEditorInstances = [];
    ckEditorSourceRecords = [];
    editors.forEach(function (editor) {
      if (editor && typeof editor.updateSourceElement === 'function') {
        editor.updateSourceElement();
      }
      resetCKEditorSourceElement(editor && editor.sourceElement);
      if (editor && typeof editor.destroy === 'function') {
        editor.destroy().catch(function () {});
      }
    });
    resetStaleCKEditorUI();
    ['item-form', 'category-form'].forEach(function (formID) {
      var form = document.getElementById(formID);
      if (form) {
        delete form.dataset.ckEditorInit;
      }
    });
  }

  function shouldUseCKEditorSourceToggle(element) {
    return element && (element.id === 'item-intro' || element.id === 'item-body');
  }

  function findCKEditorSourceRecord(element) {
    for (var i = 0; i < ckEditorSourceRecords.length; i += 1) {
      if (ckEditorSourceRecords[i].element === element) {
        return ckEditorSourceRecords[i];
      }
    }
    return null;
  }

  function removeCKEditorInstance(editor) {
    ckEditorInstances = ckEditorInstances.filter(function (item) {
      return item !== editor;
    });
  }

  function registerCKEditor(editor, element, config) {
    ckEditorInstances.push(editor);
    if (!shouldUseCKEditorSourceToggle(element)) {
      return;
    }
    var record = findCKEditorSourceRecord(element);
    if (!record) {
      record = { element: element, editor: editor, config: config, sourceMode: false };
      ckEditorSourceRecords.push(record);
    } else {
      record.editor = editor;
      record.config = config;
      record.sourceMode = false;
    }
    ensureCKEditorSourceToggle(record);
  }

  function ensureCKEditorSourceToggle(record) {
    var element = record.element;
    var field = element.closest('.admin-richtext-field');
    if (!field) {
      return;
    }
    var existing = field.querySelector('[data-ckeditor-source-toggle]');
    if (existing) {
      existing._ckEditorSourceRecord = record;
      return;
    }
    var bar = document.createElement('div');
    bar.className = 'admin-richtext-source-bar';
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'btn-admin-outline btn-admin-sm admin-richtext-source-toggle';
    button.setAttribute('data-ckeditor-source-toggle', element.id);
    button.setAttribute('aria-pressed', 'false');
    button.innerHTML = '<i class="bi bi-code-slash" aria-hidden="true"></i> <span>HTML Source</span>';
    button._ckEditorSourceRecord = record;
    button.addEventListener('click', function () {
      toggleCKEditorSourceMode(button._ckEditorSourceRecord, button);
    });
    bar.appendChild(button);
    field.insertBefore(bar, field.firstChild);
  }

  function setCKEditorSourceToggleState(button, sourceMode, busy) {
    if (!button) {
      return;
    }
    button.disabled = !!busy;
    button.classList.toggle('is-active', !!sourceMode);
    button.setAttribute('aria-pressed', sourceMode ? 'true' : 'false');
    var label = button.querySelector('span');
    if (label) {
      label.textContent = sourceMode ? 'Visual Editor' : 'HTML Source';
    }
  }

  function toggleCKEditorSourceMode(record, button) {
    if (!record || !record.element) {
      return;
    }
    var element = record.element;
    if (record.sourceMode || element.dataset.ckEditorSourceMode === '1') {
      setCKEditorSourceToggleState(button, true, true);
      loadCKEditor5().then(function (ClassicEditor) {
        return createCKEditor(ClassicEditor, element, record.config);
      }).then(function (editor) {
        element.dataset.ckEditorReady = '1';
        delete element.dataset.ckEditorSourceMode;
        element.classList.remove('admin-richtext-source-mode');
        record.editor = editor;
        record.sourceMode = false;
        registerCKEditor(editor, element, record.config);
        setCKEditorSourceToggleState(button, false, false);
      }).catch(function (err) {
        console.error('CKEditor failed to restore visual editor:', err);
        setCKEditorSourceToggleState(button, true, false);
      });
      return;
    }
    if (!record.editor || typeof record.editor.destroy !== 'function') {
      return;
    }
    setCKEditorSourceToggleState(button, false, true);
    if (typeof record.editor.updateSourceElement === 'function') {
      record.editor.updateSourceElement();
    }
    var editor = record.editor;
    editor.destroy().then(function () {
      removeCKEditorInstance(editor);
      record.editor = null;
      record.sourceMode = true;
      element.dataset.ckEditorReady = '';
      element.dataset.ckEditorSourceMode = '1';
      element.classList.add('admin-richtext-source-mode');
      element.style.display = '';
      element.hidden = false;
      setCKEditorSourceToggleState(button, true, false);
      element.focus();
    }).catch(function (err) {
      console.error('CKEditor failed to switch to source mode:', err);
      setCKEditorSourceToggleState(button, false, false);
    });
  }

  function initRichTextForm(formID, fields) {
    var form = document.getElementById(formID);
    if (!form || form.dataset.ckEditorInit) {
      return;
    }
    resetStaleCKEditorUI(form);
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
        return createCKEditor(ClassicEditor, item.element, item.config).then(function (editor) {
          if (document.getElementById(formID)) {
            registerCKEditor(editor, item.element, item.config);
          }
          return editor;
        });
      })).then(function (editors) {
        if (!document.getElementById(formID)) {
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

  function syncItemEditors() {
    syncCKEditors();
  }

  function destroyItemEditors() {
    destroyCKEditors();
  }

  function initItemEditors() {
    initRichTextForm('item-form', [
      { id: 'item-intro', key: 'intro', config: ckEditorConfig('body') },
      { id: 'item-body', key: 'body', config: ckEditorConfig('body') }
    ]);
  }

  function initCategoryEditors() {
    initRichTextForm('category-form', [
      { id: 'category-description', key: 'description', config: ckEditorConfig('intro') }
    ]);
  }

  function initExtensionRichText() {
    var root = document.querySelector('.admin-extension-app');
    if (!root) {
      return;
    }
    initLazyCKEditors(root);
    root.querySelectorAll('form').forEach(function (form) {
      if (form.dataset.ckEditorSubmitBound || !form.querySelector('textarea[data-ckeditor]')) {
        return;
      }
      form.dataset.ckEditorSubmitBound = '1';
      form.addEventListener('submit', syncCKEditors);
    });
  }

  if (!document.body.dataset.ckEditorSubmitBound) {
    document.body.dataset.ckEditorSubmitBound = '1';
    document.addEventListener('turbo:submit-start', function (e) {
      var target = e.target;
      if (target && (target.id === 'item-form' || target.id === 'category-form' || target.id === 'user-form' || (target.querySelector && target.querySelector('textarea[data-ckeditor]')))) {
        syncCKEditors();
      }
    });
  }

  var itemMediaRowsBound = false;

  function initItemMediaRows() {
    if (itemMediaRowsBound) {
      return;
    }
    itemMediaRowsBound = true;

    document.addEventListener('click', function (event) {
      var addBtn = event.target.closest('[data-repeat-add]');
      if (addBtn) {
        var form = addBtn.closest('form');
        var kind = addBtn.getAttribute('data-repeat-add');
        var list = form ? form.querySelector('[data-repeat-list="' + kind + '"]') : null;
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
        return;
      }

      var removeBtn = event.target.closest('[data-repeat-remove]');
      if (!removeBtn) {
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

  var mediaFieldEventsBound = false;

  function bindMediaFieldEvents() {
    if (mediaFieldEventsBound) {
      return;
    }
    mediaFieldEventsBound = true;
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
    if (!form) {
      return;
    }
    initSidebarTabs(form);
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

  var mediaPickerBound = false;

  function initMediaPicker() {
    if (mediaPickerBound) {
      return;
    }
    mediaPickerBound = true;

    var mediaPickerTarget = null;
    var pickerURL = '/admin/media/picker';
    var pickerType = 'images';

    function pickerDialog() {
      return document.getElementById('media-picker-dialog');
    }

    function pickerFrame() {
      return document.getElementById('media-picker-frame');
    }

    function pickerURLWithParams(folder) {
      var url = pickerURL + '?type=' + encodeURIComponent(pickerType || 'images');
      if (folder) {
        url += '&folder=' + encodeURIComponent(folder);
      }
      return url;
    }

    function openMediaPicker(target, type) {
      var dialog = pickerDialog();
      var frame = pickerFrame();
      mediaPickerTarget = target || document.getElementById('item-image');
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

    function replacePickerFrame(html) {
      var frame = pickerFrame();
      if (!frame) {
        return;
      }
      var template = document.createElement('template');
      template.innerHTML = String(html || '').trim();
      var returnedFrame = template.content.querySelector('#media-picker-frame');
      frame.innerHTML = returnedFrame ? returnedFrame.innerHTML : html;
      initMediaUpload();
      initMediaFolders();
      syncPickerUploadToggle();
    }

    function syncPickerUploadToggle(root) {
      var scope = root || pickerFrame() || document;
      var panel = scope.querySelector('[data-media-picker-upload-panel]');
      var toggle = scope.querySelector('[data-media-picker-upload-toggle]');
      if (!panel || !toggle) {
        return;
      }
      var isOpen = panel.classList.contains('is-open') && !panel.hidden;
      toggle.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
      var label = toggle.querySelector('[data-media-picker-upload-toggle-label]');
      if (label) {
        label.textContent = isOpen ? 'Hide Upload' : 'Upload';
      }
    }

    function submitPickerForm(form) {
      var frame = pickerFrame();
      if (!form || !frame) {
        return;
      }
      var submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) {
        submitBtn.disabled = true;
      }
      fetch(form.action, {
        method: form.method || 'POST',
        body: new FormData(form),
        credentials: 'same-origin',
        headers: { 'X-Requested-With': 'XMLHttpRequest' }
      }).then(function (response) {
        return response.text();
      }).then(function (html) {
        replacePickerFrame(html);
      }).catch(function () {
        if (submitBtn) {
          submitBtn.disabled = false;
        }
      });
    }

    function closeMediaPicker() {
      var dialog = pickerDialog();
      var frame = pickerFrame();
      mediaPickerTarget = null;
      if (dialog && dialog.open) {
        dialog.close();
      }
      if (frame) {
        frame.removeAttribute('src');
        frame.innerHTML = '';
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
        openMediaPicker(targetID ? document.getElementById(targetID) : document.getElementById('item-image'), 'images');
        return;
      }
      if (e.target.closest('[data-media-picker-close]')) {
        closeMediaPicker();
        return;
      }
      var dialog = pickerDialog();
      if (dialog && e.target === dialog) {
        closeMediaPicker();
        return;
      }
      var pick = e.target.closest('[data-media-pick]');
      if (pick && dialog && dialog.contains(pick)) {
        e.preventDefault();
        var target = mediaPickerTarget || document.getElementById('item-image');
        if (target) {
          target.value = pick.getAttribute('data-media-path') || '';
          syncMediaFieldPreview(target);
          target.dispatchEvent(new Event('change', { bubbles: true }));
          if (pickerType === 'files' && target.name === 'attachment_url') {
            var attachmentRow = target.closest('[data-repeat-row="attachment"]');
            var labelInput = attachmentRow ? attachmentRow.querySelector('input[name="attachment_label"]') : null;
            if (labelInput && !labelInput.value.trim()) {
              var nameEl = pick.querySelector('.admin-media-picker-name');
              var fileName = (pick.getAttribute('title') || (nameEl && nameEl.textContent) || '').trim();
              if (fileName) {
                labelInput.value = fileName;
              }
            }
          }
        }
        closeMediaPicker();
        return;
      }
      var uploadToggle = e.target.closest('[data-media-picker-upload-toggle]');
      if (uploadToggle) {
        var picker = uploadToggle.closest('[data-media-picker]');
        var panel = picker ? picker.querySelector('[data-media-picker-upload-panel]') : document.querySelector('[data-media-picker-upload-panel]');
        if (panel) {
          var isOpen = !panel.classList.contains('is-open') || panel.hidden;
          panel.classList.toggle('is-open', isOpen);
          panel.hidden = !isOpen;
          syncPickerUploadToggle(picker || document);
          if (isOpen) {
            var input = panel.querySelector('#media-file-input');
            if (input) {
              input.focus();
            }
          }
        }
        return;
      }
    });

    document.addEventListener('change', function (e) {
      var folderSelect = e.target.closest('[data-media-picker-folder]');
      var frame = pickerFrame();
      if (!folderSelect || !frame) {
        return;
      }
      var folder = folderSelect.value || '';
      frame.src = pickerURLWithParams(folder);
    });

    document.addEventListener('submit', function (e) {
      var form = e.target.closest('[data-media-picker-upload-form], [data-media-picker-folder-form]');
      if (!form) {
        return;
      }
      e.preventDefault();
      submitPickerForm(form);
    });

    document.addEventListener('close', function (e) {
      if (e.target && e.target.id === 'media-picker-dialog') {
        mediaPickerTarget = null;
        var frame = pickerFrame();
        if (frame) {
          frame.removeAttribute('src');
          frame.innerHTML = '';
        }
      }
    }, true);
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

    var form = actionSelect.form;
    if (!form) return;
    // The table is outside the bulk form so row-action forms are not nested.
    // form.elements includes its explicitly associated selection checkboxes.
    var controls = Array.prototype.slice.call(form.elements);
    var selectAll = controls.find(function (input) { return input.hasAttribute('data-select-all'); });
    var items = controls.filter(function (input) { return input.name === 'item_ids' && !input.disabled; });
    if (!selectAll) return;

    function syncSelection() {
      var selected = items.filter(function (input) { return input.checked; }).length;
      selectAll.checked = items.length > 0 && selected === items.length;
      selectAll.indeterminate = selected > 0 && selected < items.length;
      selectAll.disabled = items.length === 0;
    }

    selectAll.addEventListener('change', function () {
      items.forEach(function (input) { input.checked = selectAll.checked; });
      syncSelection();
    });
    items.forEach(function (input) { input.addEventListener('change', syncSelection); });
    syncSelection();
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

    var previewURL = null;

    function emptyState() { return document.getElementById('media-dropzone-empty'); }
    function selectedState() { return document.getElementById('media-dropzone-selected'); }
    function selectedName() { return document.getElementById('media-selected-name'); }
    function selectedSize() { return document.getElementById('media-selected-size'); }
    function selectedIcon() { return document.getElementById('media-selected-icon'); }
    function previewEmpty() { return document.getElementById('media-preview-empty'); }
    function previewImage() { return document.getElementById('media-preview-image'); }
    function submitBtn() { return document.getElementById('media-upload-submit'); }

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
      var emptyEl = emptyState();
      var selectedEl = selectedState();
      var nameEl = selectedName();
      var sizeEl = selectedSize();
      var iconEl = selectedIcon();
      var previewEmptyEl = previewEmpty();
      var previewImageEl = previewImage();
      var submitEl = submitBtn();

      if (!file) {
        setHidden(emptyEl, false);
        setHidden(selectedEl, true);
        setHidden(previewEmptyEl, false);
        setHidden(previewImageEl, true);
        if (previewImageEl) {
          previewImageEl.removeAttribute('src');
        }
        if (submitEl) {
          submitEl.disabled = true;
        }
        resetPreviewURL();
        return;
      }

      setHidden(emptyEl, true);
      setHidden(selectedEl, false);
      if (nameEl) {
        nameEl.textContent = file.name;
      }
      if (sizeEl) {
        sizeEl.textContent = formatFileSize(file.size);
      }

      var isImage = file.type && file.type.indexOf('image/') === 0;
      if (iconEl) {
        iconEl.innerHTML = isImage
          ? '<img src="" alt="" id="media-selected-thumb">'
          : '<i class="bi bi-file-earmark"></i>';
      }
      if (isImage && iconEl) {
        var thumb = document.getElementById('media-selected-thumb');
        resetPreviewURL();
        previewURL = URL.createObjectURL(file);
        if (thumb) {
          thumb.src = previewURL;
        }
        if (previewImageEl) {
          previewImageEl.src = previewURL;
        }
        setHidden(previewEmptyEl, true);
        setHidden(previewImageEl, false);
      } else {
        resetPreviewURL();
        setHidden(previewEmptyEl, false);
        setHidden(previewImageEl, true);
        if (previewImageEl) {
          previewImageEl.removeAttribute('src');
        }
      }

      if (submitEl) {
        submitEl.disabled = false;
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
        e.preventDefault();
        e.stopPropagation();
        input.value = '';
        renderFile(null);
        input.click();
        return;
      }
      input.click();
    });

    input.addEventListener('change', function () {
      handleFiles(input.files);
    });

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

  function extensionBaseName(filename) {
    var base = String(filename || '').replace(/^.*[\\/]/, '');
    var dot = base.lastIndexOf('.');
    return dot > 0 ? base.slice(0, dot) : base;
  }

  function initExtensionUpload() {
    var dropzone = document.getElementById('extension-upload-dropzone');
    var input = document.getElementById('extension-upload-file');
    if (!dropzone || !input || dropzone.dataset.bound) {
      return;
    }
    dropzone.dataset.bound = '1';

    var emptyState = document.getElementById('extension-dropzone-empty');
    var selectedState = document.getElementById('extension-dropzone-selected');
    var selectedName = document.getElementById('extension-selected-name');
    var selectedSize = document.getElementById('extension-selected-size');
    var clearBtn = document.getElementById('extension-clear-file');
    var nameInput = document.getElementById('extension-upload-name');

    function setHidden(el, hidden) {
      if (el) {
        el.classList.toggle('hidden', hidden);
      }
    }

    function renderFile(file) {
      if (!file) {
        setHidden(emptyState, false);
        setHidden(selectedState, true);
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
      if (nameInput && !String(nameInput.value || '').trim()) {
        nameInput.value = extensionBaseName(file.name);
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
      if (e.target.closest('#extension-clear-file')) {
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
      handleFiles(e.dataTransfer && e.dataTransfer.files);
    });
  }

  function copyTextFallback(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.left = '-9999px';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
    } catch (err) {}
    document.body.removeChild(ta);
  }

  function copyText(text) {
    if (!text) {
      return;
    }
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).catch(function () {
        copyTextFallback(text);
      });
      return;
    }
    copyTextFallback(text);
  }

  function initMediaCopy() {
    document.querySelectorAll('[data-copy-text]').forEach(function (btn) {
      if (btn.dataset.bound) {
        return;
      }
      btn.dataset.bound = '1';
      btn.addEventListener('click', function () {
        copyText(btn.getAttribute('data-copy-text') || '');
      });
    });
  }

  document.addEventListener('click', function (e) {
    var copyBtn = e.target.closest('[data-copy-text]');
    if (copyBtn) {
      copyText(copyBtn.getAttribute('data-copy-text') || '');
      return;
    }
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
    var navLink = e.target.closest('.admin-nav a[href]');
    if (navLink && document.body.classList.contains('admin-sidebar-open')) {
      document.body.classList.remove('admin-sidebar-open');
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
    document.querySelectorAll('form[data-ck-editor-submit-bound]').forEach(function (form) {
      delete form.dataset.ckEditorSubmitBound;
    });
    document.querySelectorAll('[data-media-rows-bound]').forEach(function (el) {
      delete el.dataset.mediaRowsBound;
    });
    var pickerDialogEl = document.getElementById('media-picker-dialog');
    if (pickerDialogEl && pickerDialogEl.open) {
      pickerDialogEl.close();
    }
    var cmdDialog = document.getElementById('admin-command-dialog');
    if (cmdDialog && cmdDialog.open) {
      cmdDialog.close();
    }
    var userMenu = document.getElementById('admin-user-menu');
    var userDropdown = document.getElementById('admin-user-menu-dropdown');
    if (userMenu) {
      userMenu.classList.remove('open');
    }
    if (userDropdown) {
      userDropdown.hidden = true;
    }
    var userToggle = document.getElementById('admin-user-menu-toggle');
    if (userToggle) {
      userToggle.setAttribute('aria-expanded', 'false');
    }
  });

  document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('turbo:load', init);
  document.addEventListener('turbo:frame-load', init);
})();
