import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { FeatureFlagsPanel } from "@/components/settings/FeatureFlagsPanel";
import { BrandingSettingsForm } from "@/components/settings/BrandingSettingsForm";
import { ApiError } from "@/lib/api/client";
import { serverApiGet } from "@/lib/api/server";
import type { FeatureFlag } from "@/types/api";

export default async function SistemaPage() {
  let flags: FeatureFlag[] | null = null;
  let forbidden = false;
  let errorMessage: string | null = null;

  try {
    const { data } = await serverApiGet<FeatureFlag[]>("v1/admin/feature-flags");
    flags = data;
  } catch (err) {
    if (err instanceof ApiError && err.status === 403) {
      forbidden = true;
    } else {
      errorMessage = err instanceof ApiError ? err.message : "Falha ao carregar feature flags";
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* 1. Branding & Identidade Visual Governamental */}
      <BrandingSettingsForm />

      {/* 2. Feature Flags do Sistema */}
      <Card>
        <CardHeader>
          <CardTitle>Feature flags & Módulos do Sistema</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-1">
          {forbidden && (
            <p className="text-sm text-muted">
              Restrito a administradores — sua conta não tem permissão para ver ou alterar feature flags.
            </p>
          )}
          {errorMessage && <ErrorState message={errorMessage} />}
          {flags && <FeatureFlagsPanel initialFlags={flags} />}
        </CardContent>
      </Card>
    </div>
  );
}
