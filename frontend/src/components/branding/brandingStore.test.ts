import { beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_BRANDING } from "./brandingConfig";

// A store é um singleton de módulo (cache em memória). vi.resetModules() +
// import dinâmico dá a cada teste uma instância zerada, como um refresh de
// página faria.
async function freshStore() {
  vi.resetModules();
  return import("./brandingStore");
}

function clearCookies() {
  for (const pair of document.cookie.split(";")) {
    const name = pair.split("=")[0]?.trim();
    if (name) document.cookie = `${name}=; path=/; max-age=0`;
  }
}

describe("brandingStore", () => {
  beforeEach(() => {
    clearCookies();
    window.localStorage.clear();
  });

  it("getServerSnapshot é o DEFAULT (fallback; o server snapshot real vem do layout)", async () => {
    const store = await freshStore();
    // toEqual, não toBe: freshStore() re-importa o módulo (e o brandingConfig
    // junto), então a referência de DEFAULT_BRANDING difere da estática aqui.
    expect(store.getBrandingServerSnapshot()).toEqual(DEFAULT_BRANDING);
  });

  it("sem cookie nem localStorage, o snapshot é o DEFAULT", async () => {
    const store = await freshStore();
    expect(store.getBrandingSnapshot()).toEqual(DEFAULT_BRANDING);
  });

  it("update mescla sobre o default e grava o cookie; snapshot reflete", async () => {
    const store = await freshStore();
    store.updateBrandingStore({ appName: "Prefeitura X", highContrast: true });
    const snap = store.getBrandingSnapshot();
    expect(snap.appName).toBe("Prefeitura X");
    expect(snap.highContrast).toBe(true);
    expect(snap.orgName).toBe(DEFAULT_BRANDING.orgName); // campos não tocados ficam

    expect(document.cookie).toContain("nova-branding=");
    const raw = decodeURIComponent(
      document.cookie.split("nova-branding=")[1]?.split(";")[0] ?? "",
    );
    expect(JSON.parse(raw).appName).toBe("Prefeitura X");
  });

  it("reset volta ao default e apaga o cookie", async () => {
    const store = await freshStore();
    store.updateBrandingStore({ appName: "Temp" });
    store.resetBrandingStore();
    expect(store.getBrandingSnapshot()).toEqual(DEFAULT_BRANDING);
    expect(document.cookie).not.toContain("nova-branding=");
  });

  it("lê um cookie pré-existente no primeiro snapshot", async () => {
    document.cookie = `nova-branding=${encodeURIComponent(
      JSON.stringify({ ...DEFAULT_BRANDING, appName: "Do Cookie" }),
    )}; path=/`;
    const store = await freshStore();
    expect(store.getBrandingSnapshot().appName).toBe("Do Cookie");
  });

  it("migra o branding legado do localStorage para cookie na 1ª leitura", async () => {
    window.localStorage.setItem(
      "nova_system_branding_v1",
      JSON.stringify({ appName: "Legado" }),
    );
    const store = await freshStore();
    expect(store.getBrandingSnapshot().appName).toBe("Legado");
    expect(document.cookie).toContain("nova-branding=");
    expect(window.localStorage.getItem("nova_system_branding_v1")).toBeNull();
  });

  it("notifica os listeners e para depois do unsubscribe", async () => {
    const store = await freshStore();
    const listener = vi.fn();
    const unsub = store.subscribeBranding(listener);

    store.updateBrandingStore({ appName: "A" });
    expect(listener).toHaveBeenCalledTimes(1);

    unsub();
    store.updateBrandingStore({ appName: "B" });
    expect(listener).toHaveBeenCalledTimes(1);
  });
});
