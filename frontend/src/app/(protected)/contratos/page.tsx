import { getServerSession } from "next-auth/next";
import { redirect } from "next/navigation";

import { authOptions } from "@/lib/auth/options";
import { serverApiGet } from "@/lib/api/server";
import { ErrorState } from "@/components/ui/ErrorState";
import { KanbanBoard } from "./KanbanBoard";
import type { KanbanResponse } from "@/types/api";
import { Button } from "@/components/ui/Button";
import { Plus } from "lucide-react";

import { getServerToken } from "@/lib/auth/serverToken";

// Server Component: carrega a visão inicial do Kanban de contratos.
export default async function ContratosPage() {
  const token = await getServerToken();
  const session = await getServerSession(authOptions);
  if (!token && !session?.user) {
    redirect("/login");
  }

  let kanbanData: KanbanResponse | null = null;
  let errorMessage: string | null = null;

  try {
    const { data } = await serverApiGet<KanbanResponse>("v1/demands/kanban");
    kanbanData = data;
  } catch (err: any) {
    errorMessage = err.message || "Falha ao carregar as demandas do Kanban";
  }

  return (
    <div className="flex h-[calc(100vh-64px)] flex-col gap-4 overflow-hidden pb-6">
      {errorMessage && <ErrorState message={errorMessage} />}

      {kanbanData && (
        <div className="flex-1 overflow-x-auto overflow-y-hidden">
          <KanbanBoard initialData={kanbanData} />
        </div>
      )}
    </div>
  );
}
