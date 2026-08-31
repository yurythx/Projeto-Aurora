"use client";

import { useState } from "react";
import useSWR from "swr";
import { Plus, RefreshCw, Box, AlertCircle } from "lucide-react";

import { Button } from "@/components/ui/Button";
import { apiClient } from "@/lib/api/client";
import { Skeleton } from "@/components/ui/Skeleton";

interface ExampleItem {
  id: string;
  title: string;
  description: string;
  created_at: string;
}

export default function ExemplosPage() {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const { data, error, isLoading, mutate } = useSWR<{ data: ExampleItem[] }>(
    "v1/example-items",
    () => apiClient.get<ExampleItem[]>("v1/example-items").then((res) => res)
  );

  const items = data?.data ?? [];

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!title.trim()) return;

    setIsCreating(true);
    setErrorMsg(null);
    try {
      await apiClient.post("v1/example-items", {
        title: title.trim(),
        description: description.trim(),
      });
      setTitle("");
      setDescription("");
      await mutate();
    } catch (err: unknown) {
      setErrorMsg(err instanceof Error ? err.message : "Erro ao criar item de exemplo.");
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Módulo Modelo (Exemplo Blueprint)</h1>
        <p className="text-sm text-muted">
          Página de referência demonstrando consumo REST tipado, gerenciamento de estado e integração com o backend do Projeto Aurora.
        </p>
      </div>

      {/* Formulário de Criação */}
      <div className="rounded-xl border border-surface-border bg-surface p-6 shadow-sm">
        <h2 className="text-base font-semibold text-foreground mb-4">Novo Item de Exemplo</h2>
        
        {errorMsg && (
          <div className="mb-4 flex items-center gap-2 rounded-lg bg-status-offline/10 p-3 text-xs text-status-offline">
            <AlertCircle size={16} />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleCreate} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-muted mb-1">Título *</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Ex: Novo serviço cadastrado"
                required
                className="w-full rounded-lg border border-surface-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-muted mb-1">Descrição</label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Ex: Descrição detalhada da demanda"
                className="w-full rounded-lg border border-surface-border bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>
          </div>

          <div className="flex justify-end">
            <Button type="submit" variant="primary" disabled={isCreating || !title.trim()}>
              <Plus size={16} className="mr-2" />
              {isCreating ? "Salvando..." : "Criar Item"}
            </Button>
          </div>
        </form>
      </div>

      {/* Tabela / Lista de Itens */}
      <div className="rounded-xl border border-surface-border bg-surface shadow-sm overflow-hidden">
        <div className="flex items-center justify-between border-b border-surface-border p-4 bg-surface-hover/50">
          <h3 className="text-sm font-medium text-foreground">Itens Cadastrados</h3>
          <Button variant="ghost" size="sm" onClick={() => mutate()}>
            <RefreshCw size={14} className="mr-1.5" />
            Atualizar
          </Button>
        </div>

        {isLoading ? (
          <div className="p-6 space-y-3">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : error ? (
          <div className="p-8 text-center text-xs text-status-offline">
            Erro ao carregar os itens de exemplo. Verifique a conexão com o backend.
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center p-12 text-center">
            <Box size={36} className="text-muted/40 mb-3" />
            <p className="text-sm font-medium text-foreground">Nenhum item cadastrado</p>
            <p className="text-xs text-muted max-w-sm mt-1">
              Utilize o formulário acima para cadastrar o primeiro registro de exemplo neste módulo.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-surface-border">
            {items.map((item) => (
              <div key={item.id} className="flex items-center justify-between p-4 hover:bg-surface-hover transition-colors">
                <div>
                  <h4 className="text-sm font-semibold text-foreground">{item.title}</h4>
                  {item.description && (
                    <p className="text-xs text-muted mt-0.5">{item.description}</p>
                  )}
                </div>
                <div className="text-[11px] font-mono text-muted">
                  {new Date(item.created_at).toLocaleTimeString("pt-BR", {
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
