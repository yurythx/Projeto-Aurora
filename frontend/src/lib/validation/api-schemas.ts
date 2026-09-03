import { z } from "zod";

// Schemas Zod das RESPOSTAS REST consumidas pelas telas — a contraparte do
// que lib/validation/schemas.ts já faz para os eventos de WebSocket (S-07
// da auditoria: "validar schema nos dois lados"). Passados opcionalmente a
// serverApiGet/apiClient; quando presentes, a resposta é validada em
// runtime antes de chegar à UI, em vez de um `as T` cego.
//
// `.passthrough()` de propósito: um campo NOVO no backend Go não deve
// quebrar o front — só a AUSÊNCIA/tipo errado de um campo que a tela usa.

export const userSchema = z
  .object({
    id: z.string(),
    username: z.string(),
    email: z.string(),
    display_name: z.string(),
    active: z.boolean(),
    created_at: z.string(),
    last_seen_at: z.string().optional(),
  })
  .passthrough();

export const usersListSchema = z.array(userSchema);

export const featureFlagSchema = z
  .object({
    key: z.string(),
    enabled: z.boolean(),
    description: z.string().optional(),
  })
  .passthrough();

export const featureFlagsListSchema = z.array(featureFlagSchema);

export const paginationMetaSchema = z
  .object({
    page: z.number(),
    page_size: z.number(),
    total_items: z.number(),
    total_pages: z.number(),
  })
  .passthrough();
