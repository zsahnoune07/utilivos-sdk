(function registerUtilivoUI(global) {
  const CORE_TOKEN_PREFIX = '--uv-';
  const localizationState = {
    defaultLocale: 'en',
    locale: 'en',
    messages: {},
    supportedLocales: ['en'],
  };

  function createElement(tagName, options) {
    const doc = global.document;
    if (!doc) {
      return null;
    }

    const config = options || {};
    const element = doc.createElement(tagName);

    if (config.className) {
      element.className = config.className;
    }
    if (config.text) {
      element.textContent = config.text;
    }
    if (config.attrs && typeof config.attrs === 'object') {
      Object.entries(config.attrs).forEach(([key, value]) => {
        if (value == null) {
          return;
        }
        element.setAttribute(key, String(value));
      });
    }
    if (Array.isArray(config.children)) {
      config.children.filter(Boolean).forEach((child) => {
        element.appendChild(child);
      });
    }

    return element;
  }

  function applyThemePayload(payload) {
    if (!payload || typeof payload !== 'object') {
      return;
    }

    const root = global.document?.documentElement;
    if (!root) {
      return;
    }

    const mode = payload.mode === 'dark' ? 'dark' : 'light';
    root.setAttribute('data-uv-theme', mode);

    const tokens = payload.tokens;
    if (!tokens || typeof tokens !== 'object') {
      return;
    }

    Object.entries(tokens).forEach(([key, value]) => {
      if (!key.startsWith(CORE_TOKEN_PREFIX)) {
        return;
      }
      if (typeof value !== 'string' || value.trim().length === 0) {
        return;
      }
      root.style.setProperty(key, value.trim());
    });
  }

  function ensureToolPage() {
    if (!global.document?.body) {
      return;
    }
    global.document.body.classList.add('uv-tool-page');
  }

  function mountToolPage(contentEl) {
    ensureToolPage();
    const body = global.document?.body;
    if (!body || !contentEl) {
      return;
    }

    const wrapper = global.document.createElement('div');
    wrapper.className = 'uv-tool-page__content';
    wrapper.appendChild(contentEl);
    body.innerHTML = '';
    body.appendChild(wrapper);
  }

  function normalizeLocale(locale) {
    return typeof locale === 'string' ? locale.trim() : '';
  }

  function resolveLocale(requestedLocale, supportedLocales, defaultLocale) {
    const requested = normalizeLocale(requestedLocale);
    if (!requested) {
      return defaultLocale;
    }
    if (supportedLocales.includes(requested)) {
      return requested;
    }

    const languageCode = requested.split('-')[0];
    if (supportedLocales.includes(languageCode)) {
      return languageCode;
    }

    const regionalMatch = supportedLocales.find(
      (locale) => locale.split('-')[0] === languageCode,
    );
    return regionalMatch || defaultLocale;
  }

  function getMessageByKey(messages, key) {
    if (!messages || typeof messages !== 'object' || typeof key !== 'string') {
      return undefined;
    }

    return key.split('.').reduce((value, part) => {
      if (!value || typeof value !== 'object') {
        return undefined;
      }
      return value[part];
    }, messages);
  }

  function interpolateMessage(message, values) {
    if (typeof message !== 'string') {
      return '';
    }
    if (!values || typeof values !== 'object') {
      return message;
    }

    return message.replace(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g, (match, key) => {
      const value = values[key];
      return value == null ? match : String(value);
    });
  }

  function translate(key, values, fallback) {
    const message = getMessageByKey(localizationState.messages, key);
    if (typeof message !== 'string') {
      return typeof fallback === 'string' ? fallback : key;
    }
    return interpolateMessage(message, values);
  }

  async function readTextResource(url) {
    if (typeof global.fetch === 'function') {
      try {
        const response = await global.fetch(url, { cache: 'no-store' });
        if (response.ok) {
          return await response.text();
        }
      } catch (_) {
        // Fall through to XMLHttpRequest for local-file runtimes.
      }
    }

    if (typeof global.XMLHttpRequest === 'function') {
      return await new Promise((resolve, reject) => {
        const request = new global.XMLHttpRequest();
        request.open('GET', url, true);
        request.onreadystatechange = () => {
          if (request.readyState !== 4) {
            return;
          }
          if ((request.status >= 200 && request.status < 300) || request.status === 0) {
            resolve(request.responseText);
            return;
          }
          reject(new Error(`Unable to load locale resource: ${url}`));
        };
        request.onerror = () => reject(new Error(`Unable to load locale resource: ${url}`));
        request.send();
      });
    }

    throw new Error(`No supported resource loader available for ${url}`);
  }

  async function loadLocaleMessages(locale, basePath, defaultLocale) {
    const uniqueLocales = Array.from(new Set([locale, defaultLocale])).filter(Boolean);

    for (const candidate of uniqueLocales) {
      const url = `${basePath}/${candidate}.json`;
      try {
        const raw = await readTextResource(url);
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          return {
            locale: candidate,
            messages: parsed,
          };
        }
      } catch (_) {
        // Keep trying until the fallback locale is exhausted.
      }
    }

    throw new Error(`Unable to load locale bundle for ${locale}`);
  }

  function applyTranslations(root) {
    const target = root || global.document;
    const doc = global.document;
    if (!target || typeof target.querySelectorAll !== 'function' || !doc) {
      return;
    }

    target.querySelectorAll('[data-i18n]').forEach((node) => {
      const key = node.getAttribute('data-i18n');
      if (!key) {
        return;
      }
      node.textContent = translate(key, undefined, node.textContent || '');
    });

    target.querySelectorAll('[data-i18n-placeholder]').forEach((node) => {
      const key = node.getAttribute('data-i18n-placeholder');
      if (!key) {
        return;
      }
      node.setAttribute(
        'placeholder',
        translate(key, undefined, node.getAttribute('placeholder') || ''),
      );
    });

    target.querySelectorAll('[data-i18n-title]').forEach((node) => {
      const key = node.getAttribute('data-i18n-title');
      if (!key) {
        return;
      }
      node.setAttribute('title', translate(key, undefined, node.getAttribute('title') || ''));
    });

    target.querySelectorAll('[data-i18n-aria-label]').forEach((node) => {
      const key = node.getAttribute('data-i18n-aria-label');
      if (!key) {
        return;
      }
      node.setAttribute(
        'aria-label',
        translate(key, undefined, node.getAttribute('aria-label') || ''),
      );
    });

    const titleKey = doc.documentElement?.getAttribute('data-i18n-document-title');
    if (titleKey) {
      doc.title = translate(titleKey, undefined, doc.title || '');
    }
    if (doc.documentElement) {
      doc.documentElement.lang = localizationState.locale;
    }
  }

  async function initLocalization(options) {
    const config = options || {};
    const supportedLocales = Array.isArray(config.supportedLocales) && config.supportedLocales.length > 0
      ? Array.from(
          new Set(
            config.supportedLocales
              .map((locale) => normalizeLocale(locale))
              .filter(Boolean),
          ),
        )
      : ['en'];
    const defaultLocale = supportedLocales.includes('en')
      ? 'en'
      : supportedLocales[0] || 'en';
    const requestedLocale =
      config.locale ||
      global.document?.documentElement?.lang ||
      global.navigator?.language ||
      defaultLocale;
    const resolvedLocale = resolveLocale(requestedLocale, supportedLocales, defaultLocale);
    const payload = await loadLocaleMessages(
      resolvedLocale,
      config.basePath || './locales',
      defaultLocale,
    );

    localizationState.defaultLocale = defaultLocale;
    localizationState.locale = payload.locale;
    localizationState.messages = payload.messages;
    localizationState.supportedLocales = supportedLocales;

    applyTranslations(config.root || global.document);

    return { ...localizationState };
  }

  function createToolCard(children, className) {
    return createElement('section', {
      className: ['uv-card', 'uv-tool-card', className].filter(Boolean).join(' '),
      children,
    });
  }

  function createToolSection(config) {
    const options = config || {};
    const children = [];

    if (options.title) {
      children.push(
        createElement('h2', {
          className: 'uv-section__title',
          text: options.title,
        }),
      );
    }
    if (options.description) {
      children.push(
        createElement('p', {
          className: 'uv-section__description',
          text: options.description,
        }),
      );
    }
    if (Array.isArray(options.children)) {
      children.push(...options.children);
    }

    return createElement('section', {
      className: ['uv-section', 'uv-tool-section', options.className]
        .filter(Boolean)
        .join(' '),
      children,
    });
  }

  function createToolButton(config) {
    const options = config || {};
    return createElement('button', {
      className: ['uv-button', options.variant && `uv-button--${options.variant}`]
        .filter(Boolean)
        .join(' '),
      text: options.label || 'Button',
      attrs: {
        type: options.type || 'button',
        disabled: options.disabled ? 'disabled' : null,
      },
    });
  }

  function createToolInput(config) {
    const options = config || {};
    return createElement('input', {
      className: 'uv-input uv-tool-input',
      attrs: {
        type: options.type || 'text',
        placeholder: options.placeholder || '',
        value: options.value || '',
      },
    });
  }

  function createToolSelect(config) {
    const options = config || {};
    const select = createElement('select', {
      className: 'uv-select uv-tool-select',
    });
    if (!select) {
      return null;
    }

    (options.options || []).forEach((optionConfig) => {
      const option = createElement('option', {
        text: optionConfig.label,
        attrs: {
          value: optionConfig.value,
          selected: optionConfig.value === options.value ? 'selected' : null,
        },
      });
      if (option) {
        select.appendChild(option);
      }
    });

    return select;
  }

  function createToolTextarea(config) {
    const options = config || {};
    return createElement('textarea', {
      className: 'uv-textarea',
      text: options.value || '',
      attrs: {
        placeholder: options.placeholder || '',
        rows: options.rows || 4,
      },
    });
  }

  function createToolResult(text) {
    return createElement('pre', {
      className: 'uv-result uv-tool-result',
      text: text || '',
    });
  }

  function createEmptyState(config) {
    const options = config || {};
    return createElement('section', {
      className: 'uv-empty-state',
      children: [
        createElement('h2', {
          className: 'uv-section__title',
          text: options.title || 'Nothing to show',
        }),
        createElement('p', {
          className: 'uv-section__description',
          text: options.message || 'No content is available yet.',
        }),
      ],
    });
  }

  function createBadge(config) {
    const options = config || {};
    return createElement('span', {
      className: ['uv-badge', options.variant && `uv-badge--${options.variant}`]
        .filter(Boolean)
        .join(' '),
      text: options.label || 'Badge',
    });
  }

  function createAlert(config) {
    const options = config || {};
    return createElement('section', {
      className: ['uv-alert', options.variant && `uv-alert--${options.variant}`]
        .filter(Boolean)
        .join(' '),
      children: [
        createElement('strong', { text: options.title || '' }),
        createElement('p', { text: options.message || '' }),
      ].filter((node) => node && node.textContent),
    });
  }

  function createToolbar(config) {
    const options = config || {};
    return createElement('header', {
      className: ['uv-toolbar', options.className].filter(Boolean).join(' '),
      children: options.children || [],
    });
  }

  function createSpinner(label) {
    return createElement('span', {
      className: 'uv-loading',
      children: [
        createElement('span', { className: 'uv-spinner', attrs: { 'aria-hidden': 'true' } }),
        createElement('span', { text: label || 'Loading...' }),
      ],
    });
  }

  function createErrorState(config) {
    const options = config || {};
    return createElement('section', {
      className: 'uv-error-state',
      children: [
        createElement('h2', {
          className: 'uv-error-state__title',
          text: options.title || 'Something went wrong',
        }),
        createElement('p', {
          className: 'uv-section__description',
          text: options.message || 'Try again once the problem is resolved.',
        }),
      ],
    });
  }

  global.UtilivoUI = {
    applyThemePayload,
    ensureToolPage,
    mountToolPage,
    initLocalization,
    applyTranslations,
    t: translate,
    getLocale: () => localizationState.locale,
    createToolButton,
    createToolCard,
    createToolInput,
    createToolResult,
    createToolSection,
    createToolSelect,
    createToolTextarea,
    createBadge,
    createAlert,
    createToolbar,
    createSpinner,
    createEmptyState,
    createErrorState,
  };
})(window);
