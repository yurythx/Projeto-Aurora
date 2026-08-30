"use client";

import { useState } from "react";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragOverlay,
  DragStartEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { updateDemandEtapa } from "@/lib/api/demands";
import { apiClient } from "@/lib/api/client";
import type { KanbanResponse, KanbanColumn, DemandResponse, StageCheck } from "@/types/api";
import { Card, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { useToast } from "@/components/notifications/ToastProvider";
import { Building2, Calendar, Lock, Plus, CheckCircle2, Clock, AlertTriangle, FileArchive, ChevronLeft, ChevronRight } from "lucide-react";

import { ContratoModal } from "./ContratoModal";
import { NewDemandModal } from "./NewDemandModal";

interface Props {
  initialData: KanbanResponse;
}

export function KanbanBoard({ initialData }: Props) {
  const [columns, setColumns] = useState<KanbanColumn[]>(initialData.columns);
  const [activeItem, setActiveItem] = useState<DemandResponse | null>(null);
  const [selectedDemand, setSelectedDemand] = useState<DemandResponse | null>(null);
  const [isNewDemandOpen, setIsNewDemandOpen] = useState(false);
  // Etapa visível no layout mobile (o quadro horizontal não cabe no celular).
  const [mobileEtapa, setMobileEtapa] = useState(1);
  const { showToast } = useToast();

  const refreshKanban = async () => {
    try {
      const { data } = await apiClient.get<KanbanResponse>("v1/demands/kanban");
      if (data && data.columns) {
        setColumns(data.columns);
      }
    } catch {
      // ignora erro silencioso de refresh
    }
  };

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 5,
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragStart = (event: DragStartEvent) => {
    const { active } = event;
    const item = columns.flatMap((col) => col.items).find((i) => i.id === active.id);
    if (item) setActiveItem(item);
  };

  // moveDemand centraliza a transição de etapa (usada pelo drag-and-drop no
  // desktop E pelos botões ←/→ do layout mobile): mesmo bloqueio preventivo,
  // mesma atualização otimista com reversão em caso de erro do backend.
  const moveDemand = async (itemId: string | number, targetEtapa: number) => {
    let sourceColIndex = -1;
    let itemToMove: DemandResponse | null = null;
    for (let i = 0; i < columns.length; i++) {
      const found = columns[i]?.items.find((it) => it.id === itemId);
      if (found) {
        sourceColIndex = i;
        itemToMove = found;
        break;
      }
    }
    if (!itemToMove) return;

    const destColIndex = columns.findIndex((c) => parseInt(c.status, 10) === targetEtapa);
    if (destColIndex === -1 || destColIndex === sourceColIndex) return;

    const sourceCol = columns[sourceColIndex];
    const destCol = columns[destColIndex];
    if (!sourceCol || !destCol) return;

    // Bloqueio PREVENTIVO: avanço de +1 com checklist da próxima etapa
    // incompleto nem chega a chamar a API — mostra o que falta.
    const req = itemToMove.next_requirements;
    if (targetEtapa === itemToMove.etapa + 1 && req && !req.wait_only && !req.can_advance) {
      showToast({
        title: "Avanço bloqueado — pendências",
        description: (req.blocking || ["Faltam documentos obrigatórios."]).join(" · "),
        tone: "danger",
      });
      return;
    }

    const prevColumns = columns;
    const newColumns = [...columns];
    newColumns[sourceColIndex] = {
      ...sourceCol,
      total: sourceCol.total - 1,
      items: sourceCol.items.filter((i) => i.id !== itemId),
    };
    const updatedItem = { ...itemToMove, etapa: targetEtapa, status_etapa: "pendente" };
    newColumns[destColIndex] = {
      ...destCol,
      total: destCol.total + 1,
      items: [updatedItem, ...destCol.items],
    };
    setColumns(newColumns);

    try {
      await updateDemandEtapa(itemToMove.id, targetEtapa);
      showToast({
        title: "Avanço Autorizado",
        description: `Demanda ${itemToMove.ano_mes} movida para ${destCol.label}.`,
        tone: "success",
      });
      refreshKanban();
    } catch (err) {
      showToast({
        title: "Avanço Bloqueado (Compliance)",
        description:
          err instanceof Error
            ? err.message
            : "Faltam documentos obrigatórios para esta etapa.",
        tone: "danger",
      });
      setColumns(prevColumns);
    }
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveItem(null);
    if (!over) return;

    let destColIndex = columns.findIndex((col) => col.status === over.id);
    if (destColIndex === -1) {
      destColIndex = columns.findIndex((col) => col.items.some((i) => i.id === over.id));
    }
    if (destColIndex === -1) return;

    await moveDemand(active.id, parseInt(columns[destColIndex]!.status, 10));
  };

  const [searchQuery, setSearchQuery] = useState("");
  const [contractTypeFilter, setContractTypeFilter] = useState<string>("TODOS");

  const displayColumns = columns.map((col) => {
    let filteredItems = col.items;

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      filteredItems = filteredItems.filter(
        (item) =>
          item.ano_mes?.toLowerCase().includes(q) ||
          item.contrato_numero?.toLowerCase().includes(q) ||
          item.contratado?.toLowerCase().includes(q) ||
          item.contrato_objeto?.toLowerCase().includes(q)
      );
    }

    if (contractTypeFilter !== "TODOS") {
      filteredItems = filteredItems.filter(
        (item) => (item as { tipo_contrato?: string }).tipo_contrato === contractTypeFilter,
      );
    }

    return {
      ...col,
      total: filteredItems.length,
      items: filteredItems,
    };
  });

  return (
    <div className="flex h-full flex-col gap-4">
      <div className="flex shrink-0 flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <p className="dateline">IN SCL 01/2019 · Demandas mensais</p>
          <h1 className="text-2xl font-semibold">Liquidação</h1>
          <p className="text-sm text-muted">
            Acompanhe cada demanda pelas seis etapas, da requisição ao arquivamento.
          </p>
        </div>
        
        <div className="flex flex-wrap items-center gap-3">
          <select
            value={contractTypeFilter}
            onChange={(e) => setContractTypeFilter(e.target.value)}
            className="h-9 rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs text-foreground focus:border-primary focus:outline-none"
          >
            <option value="TODOS">Todos os Tipos de Contrato</option>
            <option value="COMPRA_CONSUMO">Compra e Consumo</option>
            <option value="SERVICOS_TERCEIRIZADOS">Serviços Terceirizados</option>
            <option value="OBRAS">Obras e Engenharia</option>
          </select>

          <input
            type="text"
            placeholder="Filtrar por contrato, fornecedor..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-9 w-60 rounded-md border border-surface-border bg-surface px-3 py-1 text-xs text-foreground placeholder:text-muted focus:border-primary focus:outline-none"
          />
          <Button size="sm" onClick={() => setIsNewDemandOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Nova Demanda
          </Button>
        </div>
      </div>

      {/* Desktop: quadro horizontal com drag-and-drop (lg+) */}
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="hidden lg:flex flex-1 gap-4 pb-2 overflow-x-auto">
          {displayColumns.map((col) => (
            <Column key={col.status} column={col} onCardClick={setSelectedDemand} />
          ))}
        </div>

        <DragOverlay>
          {activeItem ? <DemandCard demand={activeItem} isOverlay /> : null}
        </DragOverlay>
      </DndContext>

      {/* Mobile: uma etapa por vez + botões ←/→ no lugar do arrasto */}
      <MobileBoard
        columns={displayColumns}
        currentEtapa={mobileEtapa}
        onSelectEtapa={setMobileEtapa}
        onCardClick={setSelectedDemand}
        onMove={moveDemand}
      />

      {selectedDemand && (
        <ContratoModal
          demand={selectedDemand}
          onClose={() => setSelectedDemand(null)}
          onDemandUpdated={async () => {
            await refreshKanban();
            try {
              const { data } = await apiClient.get<DemandResponse>(`v1/demands/${selectedDemand.id}`);
              if (data) {
                setSelectedDemand(data);
              }
            } catch {}
          }}
        />
      )}

      {isNewDemandOpen && (
        <NewDemandModal onClose={() => setIsNewDemandOpen(false)} onCreated={refreshKanban} />
      )}
    </div>
  );
}

function MobileBoard({
  columns,
  currentEtapa,
  onSelectEtapa,
  onCardClick,
  onMove,
}: {
  columns: KanbanColumn[];
  currentEtapa: number;
  onSelectEtapa: (etapa: number) => void;
  onCardClick: (demand: DemandResponse) => void;
  onMove: (itemId: string | number, targetEtapa: number) => void;
}) {
  const current = columns.find((c) => parseInt(c.status, 10) === currentEtapa) ?? columns[0];
  const currentNum = current ? parseInt(current.status, 10) : 1;

  return (
    <div className="lg:hidden flex flex-1 flex-col gap-3">
      {/* Seletor de etapa */}
      <div className="flex gap-2 overflow-x-auto pb-1 -mx-1 px-1">
        {columns.map((col) => {
          const n = parseInt(col.status, 10);
          const active = n === currentNum;
          return (
            <button
              key={col.status}
              onClick={() => onSelectEtapa(n)}
              className={`shrink-0 rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${
                active
                  ? "border-primary bg-primary/10 text-primary"
                  : "border-border/60 bg-surface text-muted hover:text-foreground"
              }`}
            >
              {n}. {col.label}
              <span className="ml-1.5 rounded-full bg-surface-hover px-1.5 text-[10px] text-muted">{col.total}</span>
            </button>
          );
        })}
      </div>

      {/* Cards da etapa selecionada */}
      <div className="flex flex-col gap-3">
        {current && current.items.length > 0 ? (
          current.items.map((item) => (
            <div key={item.id} className="flex flex-col gap-1.5">
              <DemandCard demand={item} onClick={() => onCardClick(item)} />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={item.etapa <= 1}
                  onClick={() => onMove(item.id, item.etapa - 1)}
                  className="flex-1 text-xs"
                >
                  <ChevronLeft className="mr-1 h-3.5 w-3.5" />
                  Etapa {item.etapa - 1}
                </Button>
                <Button
                  size="sm"
                  disabled={item.etapa >= 6}
                  onClick={() => onMove(item.id, item.etapa + 1)}
                  className="flex-1 text-xs"
                >
                  Etapa {item.etapa + 1}
                  <ChevronRight className="ml-1 h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          ))
        ) : (
          <div className="rounded-lg border border-dashed border-border/50 p-8 text-center text-xs text-muted">
            Nenhuma demanda nesta etapa.
          </div>
        )}
      </div>
    </div>
  );
}

function Column({ column, onCardClick }: { column: KanbanColumn; onCardClick: (demand: DemandResponse) => void }) {
  const itemIds = column.items.map((i) => i.id);

  return (
    <div className="flex w-80 shrink-0 flex-col rounded-lg bg-surface-hover/30 p-4 border border-border/50">
      <div className="mb-4 flex items-center justify-between gap-2">
        <h3 className="flex items-baseline gap-2 text-sm font-medium text-foreground">
          <span className="font-mono text-xs font-semibold text-seal">
            {column.status.padStart(2, "0")}
          </span>
          <span className="uppercase tracking-wider">{column.label}</span>
        </h3>
        <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-hover text-xs font-medium text-muted">
          {column.total}
        </span>
      </div>

      <div className="flex-1 overflow-y-auto overflow-x-hidden min-h-[150px] scrollbar-thin">
        <SortableContext id={column.status} items={itemIds} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-3 min-h-full">
            {column.items.map((item) => (
              <SortableDemandCard key={item.id} demand={item} onClick={() => onCardClick(item)} />
            ))}
          </div>
        </SortableContext>
      </div>
    </div>
  );
}

function SortableDemandCard({ demand, onClick }: { demand: DemandResponse; onClick: () => void }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: demand.id,
    data: {
      type: "Demand",
      demand,
    },
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners} className="cursor-grab active:cursor-grabbing outline-none">
      <DemandCard demand={demand} onClick={onClick} />
    </div>
  );
}

function DemandCard({ demand, isOverlay, onClick }: { demand: DemandResponse; isOverlay?: boolean; onClick?: () => void }) {
  return (
    <Card 
      onClick={onClick}
      className={`group relative shadow-sm transition-colors hover:border-border-hover ${isOverlay ? "rotate-2 shadow-xl scale-105" : ""} ${onClick ? "cursor-pointer" : ""}`}
    >
      <CardContent className="p-4 flex flex-col gap-3">
        <div className="flex items-start justify-between">
          <span className="inline-flex items-center rounded bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {demand.contrato_numero || "Contrato"}
          </span>
          <span className="inline-flex items-center rounded bg-surface-hover px-2 py-0.5 text-xs font-medium text-muted">
            {demand.ano_mes}
          </span>
        </div>

        <p className="text-sm font-medium leading-snug line-clamp-2" title={demand.contrato_objeto}>
          {demand.contrato_objeto || "Demanda Mensal"}
        </p>

        <div className="flex flex-col gap-1.5 pt-2 border-t border-border/40">
          <div className="flex items-center text-xs text-muted" title={demand.contratado}>
            <Building2 className="mr-1.5 h-3.5 w-3.5 shrink-0" />
            <span className="truncate">{demand.contratado || "Fornecedor"}</span>
          </div>

          <div className="flex items-center justify-between text-xs text-muted">
            <div
              className={`flex items-center ${demand.sla?.breached ? "text-danger font-medium" : ""}`}
              title={
                demand.sla?.breached && demand.sla.due_at
                  ? `SLA da etapa (${demand.sla.sla_days} dias) estourou em ${new Date(demand.sla.due_at).toLocaleDateString("pt-BR")}`
                  : demand.sla
                    ? `${demand.sla.days_in_stage}/${demand.sla.sla_days} dias na etapa`
                    : "Nesta etapa desde"
              }
            >
              <Calendar className="mr-1.5 h-3.5 w-3.5" />
              <span>
                {demand.sla ? `${demand.sla.days_in_stage}d / ${demand.sla.sla_days}d` : daysInStage(demand.etapa_started_at)}
                {demand.sla?.breached ? " · SLA!" : ""}
              </span>
            </div>
            <NextStepBadge req={demand.next_requirements} />
          </div>

          {demand.next_requirements && !demand.next_requirements.wait_only && demand.next_requirements.docs.length > 0 && (
            <div className="flex flex-wrap gap-1 pt-1" title="Documentos exigidos para a próxima etapa">
              {demand.next_requirements.docs.map((d) => (
                <span
                  key={d.doc_type}
                  title={`${d.label}${d.valid_until ? ` — válida até ${new Date(d.valid_until).toLocaleDateString("pt-BR")}` : ""}`}
                  className={`inline-block h-1.5 w-5 rounded-full ${
                    !d.present ? "bg-surface-border" : d.expired ? "bg-danger" : "bg-success"
                  }`}
                />
              ))}
            </div>
          )}

          {(demand.documents?.length ?? 0) > 0 && (
            <a
              href={`/api/backend/v1/demands/${demand.id}/package.zip`}
              download
              onClick={(e) => e.stopPropagation()}
              onPointerDown={(e) => e.stopPropagation()}
              className="mt-1 inline-flex items-center gap-1.5 self-start rounded px-1.5 py-0.5 text-xs text-muted hover:bg-surface-hover hover:text-foreground"
              title={`Baixar pacote (.zip) com os ${demand.documents?.length} documentos anexados, na ordem das etapas`}
            >
              <FileArchive className="h-3.5 w-3.5" />
              Baixar pacote ({demand.documents?.length})
            </a>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function daysInStage(startedAt: string): string {
  const d = Math.floor((Date.now() - new Date(startedAt).getTime()) / 86400000);
  if (d <= 0) return "hoje";
  return `${d}d nesta etapa`;
}

function NextStepBadge({ req }: { req?: StageCheck }) {
  if (!req) return null;
  if (req.wait_only) {
    return (
      <span className="inline-flex items-center text-amber-600 dark:text-amber-400" title="Etapa de espera externa — sem documento pendente">
        <Clock className="mr-1 h-3 w-3" /> Aguardando prazo
      </span>
    );
  }
  if (req.can_advance) {
    return (
      <span className="inline-flex items-center text-emerald-600 dark:text-emerald-400" title="Pronto para avançar">
        <CheckCircle2 className="mr-1 h-3 w-3" /> Pronto
      </span>
    );
  }
  const missing = req.docs.filter((d) => !d.present).length;
  const expired = req.docs.filter((d) => d.present && d.expired).length;
  const Icon = expired > 0 ? AlertTriangle : Lock;
  return (
    <span className="inline-flex items-center text-danger" title={(req.blocking || []).join(" · ")}>
      <Icon className="mr-1 h-3 w-3" />
      {expired > 0 ? `${expired} vencida${expired > 1 ? "s" : ""}` : `Faltam ${missing}`}
    </span>
  );
}
