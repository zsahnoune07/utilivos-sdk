(() => {
  const ui = window.UtilivoUI;
  const button = document.getElementById("ping");
  const clear = document.getElementById("clear");
  const name = document.getElementById("name");
  const tone = document.getElementById("tone");
  const output = document.getElementById("output");
  if (!ui || !button || !output || !clear || !name || !tone) return;

  async function bootstrap() {
    ui.ensureToolPage();
    await ui.initLocalization({
      defaultLocale: "en",
      supportedLocales: ["en"],
      basePath: "./locales",
    });

    const renderReady = () => {
      output.textContent = ui.t("result.ready");
    };

    renderReady();

    button.addEventListener("click", () => {
      const displayName = name.value.trim() || ui.t("result.fallback_name");
      if (tone.value === "detailed") {
        output.textContent = ui.t("result.detailed", { name: displayName });
        return;
      }
      output.textContent = ui.t("result.short", { name: displayName });
    });

    clear.addEventListener("click", () => {
      name.value = "";
      tone.value = "short";
      renderReady();
    });
  }

  bootstrap().catch((error) => {
    console.error(error);
  });
})();
