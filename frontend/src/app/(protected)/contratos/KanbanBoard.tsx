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
import type { KanbanResponse, KanbanColumn, DemandResponse } from "@/types/api";
import { Card, CardContent } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { useToast } from "@/components/notifications/ToastProvider";
import { Building2, Calendar, Lock, Plus } from "lucide-react";

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
  const { showToast } = useToast();

  const refreshKanban = async () => {
    try {
      const { data } = await apiClient.get<KanbanResponse>("/api/v1/demands/kanban");
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

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    setActiveItem(null);

    if (!over) return;

    const activeId = active.id;
    const overId = over.id;

    // Encontrar a coluna de origem
    let sourceColIndex = -1;
    let itemToMove: DemandResponse | null = null;
    
    for (let i = 0; i < columns.length; i++) {
      const col = columns[i];
      if (!col) continue;
      const item = col.items.find((it) => it.id === activeId);
      if (item) {
        sourceColIndex = i;
        itemToMove = item;
        break;
      }
    }

    if (!itemToMove) return;

    // Encontrar a coluna de destino
    let destColIndex = columns.findIndex((col) => col.status === overId);
    if (destColIndex === -1) {
      destColIndex = columns.findIndex((col) => col.items.some((i) => i.id === overId));
    }

    if (destColIndex === -1 || sourceColIndex === destColIndex) return;

    const newColumns = [...columns];
    const sourceCol = newColumns[sourceColIndex];
    const destCol = newColumns[destColIndex];
    if (!sourceCol || !destCol) return;

    // Atualiza otimisticamente a UI
    newColumns[sourceColIndex] = {
      ...sourceCol,
      total: sourceCol.total - 1,
      items: sourceCol.items.filter((i) => i.id !== activeId),
    };

    const targetEtapa = parseInt(destCol.status, 10);
    const updatedItem = { ...itemToMove, etapa: targetEtapa, status_etapa: "pendente" };
    
    newColumns[destColIndex] = {
      ...destCol,
      total: destCol.total + 1,
      items: [updatedItem, ...destCol.items],
    };

    setColumns(newColumns);

    // Persiste no backend
    try {
      await updateDemandEtapa(itemToMove.id, targetEtapa);
      showToast({
        title: "Avanço Autorizado",
        description: `Demanda ${itemToMove.ano_mes} movida para ${destCol.label}.`,
        tone: "success",
      });
    } catch (err: any) {
      // Falha na Máquina de Estados (Documentos ausentes)
      showToast({
        title: "Avanço Bloqueado (Compliance)",
        description: err.message || "Faltam documentos obrigatórios para esta etapa.",
        tone: "danger",
      });
      // Reverte a alteração otimista
      setColumns(columns);
    }
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
        (item) => (item as any).tipo_contrato === contractTypeFilter
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
        <div>
          <h1 className="text-xl font-semibold">Liquidação (Demandas Mensais)</h1>
          <p className="text-sm text-muted">Acompanhe as 6 etapas de liquidação de contratos conforme IN SCL 01/2019.</p>
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

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div className="flex flex-1 gap-4 pb-2 overflow-x-auto">
          {displayColumns.map((col) => (
            <Column key={col.status} column={col} onCardClick={setSelectedDemand} />
          ))}
        </div>

        <DragOverlay>
          {activeItem ? <DemandCard demand={activeItem} isOverlay /> : null}
        </DragOverlay>

        {selectedDemand && (
          <ContratoModal
            demand={selectedDemand}
            onClose={() => setSelectedDemand(null)}
          />
        )}

        {isNewDemandOpen && (
          <NewDemandModal
            onClose={() => setIsNewDemandOpen(false)}
            onCreated={refreshKanban}
          />
        )}
      </DndContext>
    </div>
  );
}

function Column({ column, onCardClick }: { column: KanbanColumn; onCardClick: (demand: DemandResponse) => void }) {
  const itemIds = column.items.map((i) => i.id);

  return (
    <div className="flex w-80 shrink-0 flex-col rounded-lg bg-surface-hover/30 p-4 border border-border/50">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-medium text-foreground text-sm uppercase tracking-wider">{column.label}</h3>
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-surface-hover text-xs font-medium text-muted">
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
            <div className="flex items-center">
              <Calendar className="mr-1.5 h-3.5 w-3.5" />
              <span>{new Date(demand.etapa_started_at).toLocaleDateString("pt-BR")}</span>
            </div>

            {demand.documents && demand.documents.length < 2 && demand.etapa > 1 && (
              <div className="flex items-center text-danger" title="Faltam Documentos">
                <Lock className="mr-1 h-3 w-3" />
                <span>Bloqueado</span>
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
