(() => {
  const ui = window.UtilivoUI;
  const button = document.getElementById("ping");
  const clear = document.getElementById("clear");
  const name = document.getElementById("name");
  const tone = document.getElementById("tone");
  const notes = document.getElementById("notes");
  const notify = document.getElementById("notify");
  const search = document.getElementById("search");
  const output = document.getElementById("output");
  const status = document.getElementById("status");
  const statusBody = document.getElementById("status-body");
  const list = document.getElementById("list");
  const empty = document.getElementById("empty");
  const summary = document.getElementById("summary");
  const statReady = document.getElementById("stat-ready");
  const statSearch = document.getElementById("stat-search");
  const statLocale = document.getElementById("stat-locale");
  const statTheme = document.getElementById("stat-theme");
  if (!ui || !button || !output || !clear || !name || !tone || !notes || !notify || !search || !status || !statusBody || !list || !empty || !summary || !statReady || !statSearch || !statLocale || !statTheme) return;

  const listItems = [
    { id: "calculator", title: "list.items.calculator.title", subtitle: "list.items.calculator.subtitle" },
    { id: "settings", title: "list.items.settings.title", subtitle: "list.items.settings.subtitle" },
    { id: "report", title: "list.items.report.title", subtitle: "list.items.report.subtitle" },
    { id: "quick-actions", title: "list.items.quick_actions.title", subtitle: "list.items.quick_actions.subtitle" },
  ];

  async function bootstrap() {
    ui.ensureToolPage();
    const localeState = await ui.initLocalization({
      defaultLocale: "en",
      supportedLocales: ["en"],
      basePath: "./locales",
    });

    statLocale.textContent = localeState.locale;
    statTheme.textContent = window.UtilivoHost?.theme?.resolved || "system";

    let readyCount = 0;
    let searchCount = 0;

    function setSummary(count) {
      summary.textContent = ui.t("list.summary", { count: String(count) });
      statSearch.textContent = String(searchCount);
      statReady.textContent = String(readyCount);
    }

    function renderList(query) {
      const normalizedQuery = String(query || "").trim().toLowerCase();
      const filtered = listItems.filter((item) => {
        if (!normalizedQuery) return true;
        const title = ui.t(item.title).toLowerCase();
        const subtitle = ui.t(item.subtitle).toLowerCase();
        return title.includes(normalizedQuery) || subtitle.includes(normalizedQuery);
      });

      list.innerHTML = "";
      empty.hidden = filtered.length > 0;
      list.hidden = filtered.length === 0;

      filtered.forEach((item) => {
        const row = document.createElement("button");
        row.className = "uv-list__row uv-interactive";
        row.type = "button";
        row.setAttribute("role", "option");
        row.setAttribute("aria-selected", "false");

        const content = document.createElement("div");
        content.className = "uv-stack uv-stack--sm";
        const title = document.createElement("strong");
        title.textContent = ui.t(item.title);
        const subtitle = document.createElement("span");
        subtitle.className = "uv-text-body-sm uv-text-muted";
        subtitle.textContent = ui.t(item.subtitle);
        content.appendChild(title);
        content.appendChild(subtitle);

        const badge = ui.createBadge({ variant: "info", label: ui.t("list.badge") });
        row.appendChild(content);
        row.appendChild(badge);

        row.addEventListener("click", () => {
          Array.from(list.querySelectorAll(".uv-list__row")).forEach((node) => {
            node.setAttribute("aria-selected", "false");
            node.classList.remove("uv-is-selected");
          });
          row.setAttribute("aria-selected", "true");
          row.classList.add("uv-is-selected");
          statusBody.textContent = ui.t("list.selected", { item: ui.t(item.title) });
          status.hidden = false;
        });

        list.appendChild(row);
      });

      setSummary(filtered.length);
    }

    const renderReady = () => {
      output.textContent = ui.t("result.ready");
      status.hidden = true;
      readyCount += 1;
      setSummary(listItems.length);
    };

    renderList("");
    renderReady();

    button.addEventListener("click", () => {
      const displayName = name.value.trim() || ui.t("result.fallback_name");
      const noteValue = notes.value.trim();
      const notifyState = notify.checked ? ui.t("result.notify.on") : ui.t("result.notify.off");
      if (tone.value === "detailed") {
        output.textContent = ui.t("result.detailed", {
          name: displayName,
          notes: noteValue || ui.t("result.notes.empty"),
          notify: notifyState,
        });
        return;
      }
      output.textContent = ui.t("result.short", {
        name: displayName,
        notify: notifyState,
      });
    });

    clear.addEventListener("click", () => {
      name.value = "";
      tone.value = "short";
      notes.value = "";
      notify.checked = false;
      search.value = "";
      renderList("");
      renderReady();
    });

    search.addEventListener("input", () => {
      searchCount += 1;
      renderList(search.value);
    });
  }

  bootstrap().catch((error) => {
    console.error(error);
  });
})();
