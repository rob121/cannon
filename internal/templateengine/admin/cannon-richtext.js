/**
 * Cannon rich text (CKEditor 5) — self-hosted npm UMD (ckeditor5@41.4.2 dist/browser).
 * Custom plugins require window.CKEDITOR exports; do not use CDN prebuilt Classic.
 */
(function (global) {
  'use strict';

  var CKEDITOR5_UMD_URL = '/admin/assets/ckeditor5.umd.js?v=2';
  var CKEDITOR5_CSS_URL = '/admin/assets/ckeditor5.css?v=2';

  var ckEditorLoadPromise = null;
  var ckEditorInstances = [];
  var cachedRichTextSettings = null;

  var defaultRichTextSettings = {
    enterKeyMode: 'paragraph',
    paragraphSpacing: 'default',
    introToolbar: [
      'bold', 'italic', 'link', '|',
      'bulletedList', 'numberedList', '|',
      'cannonMediaImage', '|',
      'sourceEditing', '|', 'undo', 'redo'
    ],
    bodyToolbar: [
      'heading', '|',
      'bold', 'italic', 'link', '|',
      'bulletedList', 'numberedList', 'blockQuote', '|',
      'cannonMediaImage', 'insertTable', 'mediaEmbed', '|',
      'sourceEditing', '|', 'undo', 'redo'
    ]
  };

  function loadRichTextSettings() {
    if (cachedRichTextSettings) {
      return cachedRichTextSettings;
    }
    var merged = {
      enterKeyMode: defaultRichTextSettings.enterKeyMode,
      paragraphSpacing: defaultRichTextSettings.paragraphSpacing,
      introToolbar: defaultRichTextSettings.introToolbar.slice(),
      bodyToolbar: defaultRichTextSettings.bodyToolbar.slice()
    };
    var el = document.getElementById('cannon-richtext-settings');
    if (el && el.textContent) {
      try {
        var parsed = JSON.parse(el.textContent);
        if (parsed.enterKeyMode) {
          merged.enterKeyMode = parsed.enterKeyMode;
        }
        if (parsed.paragraphSpacing) {
          merged.paragraphSpacing = parsed.paragraphSpacing;
        }
        if (parsed.introToolbar && parsed.introToolbar.length) {
          merged.introToolbar = parsed.introToolbar;
        }
        if (parsed.bodyToolbar && parsed.bodyToolbar.length) {
          merged.bodyToolbar = parsed.bodyToolbar;
        }
      } catch (e) {
        console.warn('Cannon rich text settings JSON invalid', e);
      }
    }
    cachedRichTextSettings = merged;
    return merged;
  }

  function buildEnterModePlugin(mode) {
    if (mode !== 'line_break' || !global.CKEDITOR || !global.CKEDITOR.Plugin) {
      return null;
    }
    var Plugin = global.CKEDITOR.Plugin;
    return class CannonEnterMode extends Plugin {
      static get pluginName() {
        return 'CannonEnterMode';
      }
      init() {
        var editor = this.editor;
        this.listenTo(editor.editing.view.document, 'enter', function (evt, data) {
          data.preventDefault();
          evt.stop();
          if (data.isSoft) {
            editor.execute('enter');
          } else {
            editor.execute('shiftEnter');
          }
          editor.editing.view.scrollToTheSelection();
        }, { priority: 'high' });
      }
    };
  }

  function applyRichTextFieldClasses(element) {
    var settings = loadRichTextSettings();
    var wrap = element.closest('.admin-richtext-field') || element.parentElement;
    if (!wrap) {
      return;
    }
    if (settings.paragraphSpacing === 'compact') {
      wrap.classList.add('ck-richtext-compact');
    } else {
      wrap.classList.remove('ck-richtext-compact');
    }
  }

  function richtextKindFromKey(key) {
    return key === 'body' ? 'body' : 'intro';
  }

  function richtextKindFromElement(el) {
    var key = el.getAttribute('data-ckeditor') || el.id || '';
    return richtextKindFromKey(key);
  }

  function ensureCKEditor5Stylesheet() {
    if (document.querySelector('link[data-cannon-ckeditor5-css]')) {
      return;
    }
    var link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = CKEDITOR5_CSS_URL;
    link.setAttribute('data-cannon-ckeditor5-css', '1');
    document.head.appendChild(link);
  }

  function getCKEditor5Constructor() {
    if (global.CKEDITOR && global.CKEDITOR.ClassicEditor && global.CKEDITOR.GeneralHtmlSupport) {
      return global.CKEDITOR.ClassicEditor;
    }
    return null;
  }

  function loadCKEditor5() {
    ensureCKEditor5Stylesheet();
    var editorCtor = getCKEditor5Constructor();
    if (editorCtor) {
      return Promise.resolve(editorCtor);
    }
    if (ckEditorLoadPromise) {
      return ckEditorLoadPromise;
    }
    ckEditorLoadPromise = new Promise(function (resolve, reject) {
      var script = document.createElement('script');
      script.src = CKEDITOR5_UMD_URL;
      script.async = true;
      script.onload = function () {
        var ctor = getCKEditor5Constructor();
        if (ctor) {
          resolve(ctor);
        } else {
          reject(new Error('CKEditor UMD failed to load'));
        }
      };
      script.onerror = function () {
        reject(new Error('CKEditor UMD failed to load'));
      };
      document.head.appendChild(script);
    });
    return ckEditorLoadPromise;
  }

  function createCKEditor(ClassicEditor, element, kind) {
    var config = ckEditorConfig(kind);
    if (!config.plugins) {
      return Promise.reject(new Error('CKEditor plugins unavailable; UMD may not have loaded'));
    }
    return ClassicEditor.create(element, config);
  }

  function toolbarNeeds(toolbar, name) {
    for (var i = 0; i < toolbar.length; i += 1) {
      if (toolbar[i] === name) {
        return true;
      }
    }
    return false;
  }

  function mediaPickerAvailable() {
    return !!document.getElementById('media-picker-dialog');
  }

  function toolbarForPage(toolbar) {
    if (mediaPickerAvailable()) {
      return toolbar;
    }
    return toolbar.filter(function (item) {
      return item !== 'cannonMediaImage';
    });
  }

  function pushUnique(plugins, item) {
    if (!item) {
      return;
    }
    if (plugins.indexOf(item) === -1) {
      plugins.push(item);
    }
  }

  function imageFeaturesEnabled(toolbar) {
    return toolbarNeeds(toolbar, 'uploadImage') ||
      toolbarNeeds(toolbar, 'insertImage') ||
      toolbarNeeds(toolbar, 'cannonMediaImage');
  }

  function appendImagePlugins(plugins, c, toolbar) {
    if (!imageFeaturesEnabled(toolbar)) {
      return;
    }
    pushUnique(plugins, c.ImageBlock);
    pushUnique(plugins, c.ImageInsert);
    pushUnique(plugins, c.ImageToolbar);
    pushUnique(plugins, c.ImageCaption);
    pushUnique(plugins, c.ImageStyle);
    if (toolbarNeeds(toolbar, 'uploadImage') || toolbarNeeds(toolbar, 'insertImage')) {
      pushUnique(plugins, c.ImageUpload);
      pushUnique(plugins, c.Base64UploadAdapter);
    }
  }

  function buildCannonMediaImagePlugin(c) {
    if (!c.Plugin || !c.ButtonView) {
      return null;
    }
    if (!document.getElementById('media-picker-dialog')) {
      return null;
    }
    var Plugin = c.Plugin;
    var ButtonView = c.ButtonView;
    return class CannonMediaImage extends Plugin {
      static get pluginName() {
        return 'CannonMediaImage';
      }
      init() {
        var editor = this.editor;
        editor.ui.componentFactory.add('cannonMediaImage', function (locale) {
          var view = new ButtonView(locale);
          view.set({
            label: 'Media library',
            tooltip: true
          });
          view.on('execute', function () {
            var picker = global.CannonMediaPicker;
            if (!picker || typeof picker.open !== 'function') {
              console.warn('Cannon media picker is not available on this page');
              return;
            }
            picker.open({
              type: 'images',
              onPick: function (path) {
                if (!path) {
                  return;
                }
                editor.execute('insertImage', { source: path });
              }
            });
          });
          return view;
        });
      }
    };
  }

  function appendToolbarPlugins(plugins, c, toolbar) {
    if (toolbarNeeds(toolbar, 'heading')) {
      pushUnique(plugins, c.Heading);
    }
    if (toolbarNeeds(toolbar, 'blockQuote')) {
      pushUnique(plugins, c.BlockQuote);
    }
    if (toolbarNeeds(toolbar, 'insertTable') && c.Table) {
      pushUnique(plugins, c.Table);
      pushUnique(plugins, c.TableToolbar);
    }
    if (toolbarNeeds(toolbar, 'mediaEmbed')) {
      pushUnique(plugins, c.MediaEmbed);
    }
    appendImagePlugins(plugins, c, toolbar);
    if (toolbarNeeds(toolbar, 'cannonMediaImage')) {
      var mediaPlugin = buildCannonMediaImagePlugin(c);
      if (mediaPlugin) {
        pushUnique(plugins, mediaPlugin);
      }
    }
    if (toolbarNeeds(toolbar, 'sourceEditing')) {
      pushUnique(plugins, c.SourceEditing);
    }
  }

  function ckEditorPlugins(kind) {
    var c = global.CKEDITOR;
    if (!c || !c.GeneralHtmlSupport) {
      return null;
    }
    var settings = loadRichTextSettings();
    var toolbar = toolbarForPage(kind === 'body' ? settings.bodyToolbar : settings.introToolbar);

    var plugins = [
      c.Essentials,
      c.Paragraph,
      c.GeneralHtmlSupport,
      c.Bold,
      c.Italic,
      c.Link,
      c.List
    ];
    appendToolbarPlugins(plugins, c, toolbar);
    var enterPlugin = buildEnterModePlugin(settings.enterKeyMode);
    if (enterPlugin) {
      plugins.push(enterPlugin);
    }
    return plugins.filter(Boolean);
  }

  function ckEditorConfig(kind) {
    var settings = loadRichTextSettings();
    var toolbar = toolbarForPage(
      kind === 'body' ? settings.bodyToolbar.slice() : settings.introToolbar.slice()
    );

    var config = {
      toolbar: toolbar
    };
    var plugins = ckEditorPlugins(kind);
    if (!plugins) {
      return config;
    }

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
    if (toolbarNeeds(toolbar, 'insertTable')) {
      config.table = {
        contentToolbar: ['tableColumn', 'tableRow', 'mergeTableCells']
      };
    }
    if (imageFeaturesEnabled(toolbar)) {
      config.image = {
        toolbar: [
          'imageTextAlternative', '|',
          'imageStyle:inline', 'imageStyle:block', 'imageStyle:side'
        ]
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

  function resetCKEditorElement(element) {
    if (!element) {
      return;
    }
    delete element.dataset.ckEditorReady;
    element.classList.remove('ck-hidden');
    element.hidden = false;
    element.style.display = '';

    var field = element.closest('.admin-richtext-field');
    if (field) {
      field.querySelectorAll('.ck.ck-editor').forEach(function (wrapper) {
        wrapper.remove();
      });
    } else {
      var parent = element.parentElement;
      if (parent) {
        parent.querySelectorAll('.ck.ck-editor').forEach(function (wrapper) {
          wrapper.remove();
        });
      }
    }
  }

  function resetStaleCKEditorUI(root) {
    var scope = root || document;
    scope.querySelectorAll('textarea[data-ckeditor]').forEach(resetCKEditorElement);
  }

  function destroyCKEditors() {
    var editors = ckEditorInstances.slice();
    ckEditorInstances = [];
    editors.forEach(function (editor) {
      if (editor && typeof editor.updateSourceElement === 'function') {
        editor.updateSourceElement();
      }
      resetCKEditorElement(editor && editor.sourceElement);
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

  function registerCKEditor(editor) {
    ckEditorInstances.push(editor);
  }

  function refreshCKEditors() {
    ckEditorInstances.forEach(function (editor) {
      if (editor && editor.ui && typeof editor.ui.update === 'function') {
        editor.ui.update();
      }
    });
    global.dispatchEvent(new Event('resize'));
  }

  function initLazyCKEditors(container) {
    if (!container) {
      return;
    }
    var pending = [];
    container.querySelectorAll('[data-ckeditor]').forEach(function (el) {
      if (el.dataset.ckEditorReady && !findCKEditorInstanceForElement(el)) {
        resetCKEditorElement(el);
      }
      if (!el.id || el.dataset.ckEditorReady) {
        return;
      }
      pending.push({ element: el, kind: richtextKindFromElement(el) });
    });
    if (!pending.length) {
      return;
    }
    loadCKEditor5().then(function (ClassicEditor) {
      return Promise.all(pending.map(function (item) {
        if (item.element.dataset.ckEditorReady) {
          return null;
        }
        return createCKEditor(ClassicEditor, item.element, item.kind).then(function (editor) {
          item.element.dataset.ckEditorReady = '1';
          applyRichTextFieldClasses(item.element);
          registerCKEditor(editor);
          return editor;
        });
      }));
    }).then(function () {
      refreshCKEditors();
    }).catch(function (err) {
      console.error('CKEditor lazy init failed:', err);
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
        var kind = field.kind || richtextKindFromKey(field.key);
        configs.push({ element: el, kind: kind });
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
        return createCKEditor(ClassicEditor, item.element, item.kind).then(function (editor) {
          if (document.getElementById(formID)) {
            item.element.dataset.ckEditorReady = '1';
            applyRichTextFieldClasses(item.element);
            registerCKEditor(editor);
          }
          return editor;
        });
      }));
    }).catch(function (err) {
      console.error('CKEditor failed to initialize:', err);
      delete form.dataset.ckEditorInit;
    });
  }

  /** Initialize all [data-ckeditor] fields under root (e.g. frontend content edit). */
  function initRichTextFields(root) {
    var scope = root || document;
    var elements = scope.querySelectorAll('textarea[data-ckeditor]');
    if (!elements.length) {
      return Promise.resolve();
    }
    var pending = [];
    elements.forEach(function (el) {
      if (el.dataset.ckEditorReady) {
        return;
      }
      if (!el.id) {
        el.id = 'cannon-richtext-' + Math.random().toString(36).slice(2, 10);
      }
      pending.push({ element: el, kind: richtextKindFromElement(el) });
    });
    if (!pending.length) {
      return Promise.resolve();
    }
    return loadCKEditor5().then(function (ClassicEditor) {
      return Promise.all(pending.map(function (item) {
        return createCKEditor(ClassicEditor, item.element, item.kind).then(function (editor) {
          item.element.dataset.ckEditorReady = '1';
          applyRichTextFieldClasses(item.element);
          registerCKEditor(editor);
          return editor;
        });
      }));
    });
  }

  function reloadRichTextSettings() {
    cachedRichTextSettings = null;
    loadRichTextSettings();
  }

  function bindRichTextFormSubmit(form) {
    if (!form || form.dataset.ckEditorSubmitBound) {
      return;
    }
    if (!form.querySelector('textarea[data-ckeditor]')) {
      return;
    }
    form.dataset.ckEditorSubmitBound = '1';
    form.addEventListener('submit', syncCKEditors);
  }

  global.CannonRichText = {
    loadCKEditor5: loadCKEditor5,
    ckEditorConfig: ckEditorConfig,
    createCKEditor: createCKEditor,
    syncCKEditors: syncCKEditors,
    destroyCKEditors: destroyCKEditors,
    initLazyCKEditors: initLazyCKEditors,
    initRichTextForm: initRichTextForm,
    initRichTextFields: initRichTextFields,
    refreshCKEditors: refreshCKEditors,
    resetCKEditorElement: resetCKEditorElement,
    resetStaleCKEditorUI: resetStaleCKEditorUI,
    bindRichTextFormSubmit: bindRichTextFormSubmit,
    loadRichTextSettings: loadRichTextSettings,
    reloadRichTextSettings: reloadRichTextSettings
  };
})(typeof window !== 'undefined' ? window : global);
