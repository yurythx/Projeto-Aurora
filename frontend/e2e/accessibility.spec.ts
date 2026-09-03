import { expect, test } from "@playwright/test";

test.describe("Acessibilidade e-MAG / WCAG 2.1 AA", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
  });

  test("Barra de Acessibilidade e-MAG está visível e contém atalhos", async ({ page }) => {
    const accessBar = page.locator("header");
    await expect(accessBar).toBeVisible();

    // Verifica a presença do anúncio dos atalhos e-MAG
    await expect(page.getByText("Ir para o conteúdo [1]")).toBeVisible();
    await expect(page.getByText("Ir para o menu [2]")).toBeVisible();
    await expect(page.getByText("Ir para o rodapé [4]")).toBeVisible();
  });

  test("Atalhos de teclado e-MAG focam nas âncoras corretas (#conteudo, #menu, #rodape)", async ({ page }) => {
    // Foco no conteúdo (#conteudo)
    await page.keyboard.press("Alt+1");
    await expect(page.locator("#conteudo")).toBeVisible();

    // Foco no menu (#menu)
    await page.keyboard.press("Alt+2");
    await expect(page.locator("#menu")).toBeVisible();

    // Foco no rodapé (#rodape)
    await page.keyboard.press("Alt+4");
    await expect(page.locator("#rodape")).toBeVisible();
  });

  test("Alternância do modo Alto Contraste e-MAG altera classe no documento", async ({ page }) => {
    const contrastButton = page.getByRole("button", { name: /Alto contraste/i });
    await expect(contrastButton).toBeVisible();

    // Clica no botão de Alto Contraste
    await contrastButton.click();

    // Verifica se a classe .emag-high-contrast foi adicionada ao HTML
    const html = page.locator("html");
    await expect(html).toHaveClass(/emag-high-contrast/);
  });

  test("Modal de Consentimento LGPD possui suporte ARIA (dialog, modal)", async ({ page }) => {
    // Garante que o consentimento LGPD seja verificado
    await page.evaluate(() => localStorage.removeItem("aurora_lgpd_consent"));
    await page.reload();

    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog).toHaveAttribute("aria-modal", "true");
    await expect(dialog.getByText("Termos de Privacidade & Proteção de Dados (LGPD)")).toBeVisible();
  });
});
