-- Answer bank for Krill Billion.
-- Apply once in the Supabase SQL editor. Re-running is safe: tables are
-- created if missing, and the view and upsert function are replaced.
--
-- The Go server reads SUPABASE_URL plus either SUPABASE_DB_URL (Postgres)
-- or SUPABASE_SERVICE_ROLE_KEY (PostgREST). Those values stay in the
-- environment / Fly secrets. This file has no credentials.
--
-- Rarity is per prompt. The same entity can be common for one question
-- and niche for another. Requires Postgres 15 (Supabase).

create table if not exists public.categories (
  id text primary key check (char_length(id) > 0),
  name text not null unique check (char_length(name) > 0)
);

create table if not exists public.prompts (
  id text primary key check (char_length(id) > 0),
  category_id text not null references public.categories (id),
  text text not null check (char_length(text) > 0)
);

create table if not exists public.entities (
  id text primary key check (char_length(id) > 0),
  label text not null check (char_length(label) > 0)
);

create table if not exists public.aliases (
  entity_id text not null references public.entities (id) on delete cascade,
  alias text not null check (char_length(alias) > 0),
  primary key (entity_id, alias)
);

create table if not exists public.prompt_entities (
  prompt_id text not null references public.prompts (id) on delete cascade,
  entity_id text not null references public.entities (id),
  rarity integer not null check (rarity >= 0 and rarity <= 100),
  primary key (prompt_id, entity_id)
);

alter table public.categories enable row level security;
alter table public.prompts enable row level security;
alter table public.entities enable row level security;
alter table public.aliases enable row level security;
alter table public.prompt_entities enable row level security;

-- No policies: anon and authenticated see nothing. The table owner and
-- the service role bypass RLS.

grant select, insert, update, delete on
  public.categories,
  public.prompts,
  public.entities,
  public.aliases,
  public.prompt_entities
  to service_role;

drop view if exists public.answer_bank;
create view public.answer_bank
with (security_invoker = false) as
select
  p.id as prompt_id,
  p.text as prompt_text,
  c.name as category_name,
  e.id as entity_id,
  e.label as label,
  pe.rarity as rarity,
  a.alias as alias
from public.prompts p
join public.categories c on c.id = p.category_id
join public.prompt_entities pe on pe.prompt_id = p.id
join public.entities e on e.id = pe.entity_id
left join public.aliases a on a.entity_id = e.id;

revoke all on public.answer_bank from public;
grant select on public.answer_bank to service_role;

create or replace function public.upsert_answer_bank(payload jsonb)
returns void
language plpgsql
security definer
set search_path = public
as $$
declare
  prompt_ids text[];
  entity_ids text[];
begin
  if payload is null then
    raise exception 'payload is required';
  end if;

  insert into categories (id, name)
  select c->>'id', c->>'name'
  from jsonb_array_elements(coalesce(payload->'categories', '[]'::jsonb)) c
  on conflict (id) do update set name = excluded.name;

  insert into entities (id, label)
  select e->>'id', e->>'label'
  from jsonb_array_elements(coalesce(payload->'entities', '[]'::jsonb)) e
  on conflict (id) do update set label = excluded.label;

  insert into prompts (id, category_id, text)
  select p->>'id', p->>'category_id', p->>'text'
  from jsonb_array_elements(coalesce(payload->'prompts', '[]'::jsonb)) p
  on conflict (id) do update
    set category_id = excluded.category_id,
        text = excluded.text;

  select coalesce(array_agg(distinct p->>'id'), '{}')
    into prompt_ids
  from jsonb_array_elements(coalesce(payload->'prompts', '[]'::jsonb)) p;

  select coalesce(array_agg(distinct e->>'id'), '{}')
    into entity_ids
  from jsonb_array_elements(coalesce(payload->'entities', '[]'::jsonb)) e;

  delete from prompt_entities where prompt_id = any(prompt_ids);
  delete from aliases where entity_id = any(entity_ids);

  insert into aliases (entity_id, alias)
  select distinct a->>'entity_id', a->>'alias'
  from jsonb_array_elements(coalesce(payload->'aliases', '[]'::jsonb)) a
  where coalesce(a->>'alias', '') <> '';

  insert into prompt_entities (prompt_id, entity_id, rarity)
  select pe->>'prompt_id', pe->>'entity_id', (pe->>'rarity')::integer
  from jsonb_array_elements(coalesce(payload->'prompt_entities', '[]'::jsonb)) pe;
end;
$$;

revoke all on function public.upsert_answer_bank(jsonb) from public;
grant execute on function public.upsert_answer_bank(jsonb) to service_role;
