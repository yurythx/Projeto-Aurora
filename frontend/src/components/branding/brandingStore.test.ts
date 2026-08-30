import { beforeEach, describe, expect, it, vi } from "vitest";

import { DEFAULT_BRANDING } from "./brandingConfig";
import {
  getBrandingServerSnapshot,
  getBrandingSnapshot,
  resetBrandingStore,
  subscribeBranding,
  updateBrandingStore,
} from "./brandingStore";

describe("brandingStore", () => {
  beforeEach(() => {
    window.localStorage.clear();
    // O módulo cacheia em memória entre chamadas no mesmo processo de
    // teste — reset força a releitura do localStorage limpo acima.
    resetBrandingStore();
  });

  it("getServerSnapshot é sempre o DEFAULT_BRANDING (SSR não tem branding do dispositivo)", () => {
    expect(getBrandingServerSnapshot()).toBe(DEFAULT_BRANDING);
  });

  it("sem nada no localStorage, o snapshot é o DEFAULT", () => {
    expect(getBrandingSnapshot()).toEqual(DEFAULT_BRANDING);
  });

  it("update mescla sobre o default e persiste; snapshot reflete", () => {
    updateBrandingStore({ appName: "Prefeitura X", highContrast: true });
    const snap = getBrandingSnapshot();
    expect(snap.appName).toBe("Prefeitura X");
    expect(snap.highContrast).toBe(true);
    expect(snap.orgName).toBe(DEFAULT_BRANDING.orgName); // campos não tocados ficam

    const raw = JSON.parse(window.localStorage.getItem("nova_system_branding_v1")!);
    expect(raw.appName).toBe("Prefeitura X");
  });

  it("reset volta ao default e limpa o localStorage", () => {
    updateBrandingStore({ appName: "Temp" });
    resetBrandingStore();
    expect(getBrandingSnapshot()).toEqual(DEFAULT_BRANDING);
    expect(window.localStorage.getItem("nova_system_branding_v1")).toBeNull();
  });

  it("notifica os listeners e para depois de cancelar a inscrição", () => {
    const listener = vi.fn();
    const unsub = subscribeBranding(listener);

    updateBrandingStore({ appName: "A" });
    expect(listener).toHaveBeenCalledTimes(1);

    unsub();
    updateBrandingStore({ appName: "B" });
    expect(listener).toHaveBeenCalledTimes(1);
  });
});
