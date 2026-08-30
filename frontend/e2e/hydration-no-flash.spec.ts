import { expect, test } from "@playwright/test";

// Regressão de "flash na hidratação": o nome da aplicação (branding
// white-label) e o estado recolhido da Sidebar são preferências de
// dispositivo. Enquanto viviam só em localStorage, todo refresh renderizava
// o HTML do servidor com os PADRÕES (nome "Projeto Nova", Sidebar
// expandida) e o useSyncExternalStore trocava pelos valores salvos logo
// após hidratar — a "piscada" que o usuário via.
//
// O fix padroniza no mesmo mecanismo do tema: cookie lido no layout do
// servidor → prop initial* → server snapshot do useSyncExternalStore. Este
// teste prova que o HTML CRU do servidor (pré-hidratação) já vem no estado
// certo, então não há troca visível depois.
//
// Mesmo gate da suíte de auth: sem E2E_ADMIN_PASSWORD (via `make
// seed-admin`), pula.
const ADMIN_USERNAME = process.env.E2E_ADMIN_USERNAME ?? "admin";
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD;

test.describe("Sem flash de hidratação (branding + Sidebar)", () => {
  test.skip(
    !ADMIN_PASSWORD,
    "E2E_ADMIN_PASSWORD não definida — rode `make seed-admin` e exporte a senha impressa (ver README, seção Testes).",
  );

  test("o HTML do servidor já traz o nome novo, a Sidebar recolhida e o alto contraste", async ({
    page,
    context,
  }) => {
    // --- login local ---
    await page.goto("/login");
    await page.getByLabel("Usuário").fill(ADMIN_USERNAME);
    await page.getByLabel("Senha", { exact: true }).fill(ADMIN_PASSWORD!);
    await page.getByRole("button", { name: "Entrar", exact: true }).click();
    await expect(page).toHaveURL(/\/dashboard$/);

    // --- troca o nome da aplicação em /configuracao ---
    const newName = `Fiscaliza Rondon ${Date.now().toString().slice(-5)}`;
    await page.goto("/configuracao");
    const nameField = page.getByPlaceholder("Ex: Projeto Nova");
    await nameField.fill(newName);
    await page.getByRole("button", { name: /Salvar Altera/i }).click();
    await expect(page.getByText("Configurações Salvas")).toBeVisible();

    // --- recolhe a Sidebar e liga o Alto Contraste ---
    await page.goto("/dashboard");
    await page.getByRole("button", { name: "Alternar menu lateral" }).click();
    await page.getByRole("button", { name: /Alternar Modo Alto Contraste/i }).click();
    // deixa os document.cookie assentarem
    await expect
      .poll(async () => (await context.cookies()).map((c) => c.name))
      .toEqual(expect.arrayContaining(["nova-branding", "nova-sidebar-collapsed"]));

    // --- HTML CRU do servidor: o que chega ANTES do React hidratar ---
    const html = await (await context.request.get("/dashboard")).text();

    // newName é único e só pode ter vindo de branding.appName (cookie) —
    // se o SSR tivesse renderizado o default, ele estaria ausente. (O
    // <title> estático do layout raiz ainda diz "Projeto Nova"; isso é a
    // aba do navegador, não a barra superior, e não pisca.)
    expect(html, "SSR deve trazer o nome novo — senão a Topbar pisca").toContain(newName);

    // Sidebar recolhida usa a largura recolhida (--sidebar-w-collapsed) no
    // <nav> e no padding-left do conteúdo; expandida usaria --sidebar-w.
    expect(html, "SSR deve trazer a Sidebar recolhida").toContain(
      "md:w-[var(--sidebar-w-collapsed)]",
    );
    expect(html).not.toContain("md:w-[var(--sidebar-w)]");
    expect(html).toContain("md:pl-[var(--sidebar-w-collapsed)]");

    expect(html, "SSR deve carimbar data-high-contrast no <html>").toMatch(
      /<html[^>]*data-high-contrast="true"/,
    );
  });
});
