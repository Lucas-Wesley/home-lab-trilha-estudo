#!/usr/bin/env python3
"""Revisor automático de uma trilha (regras/criar-trilha.md, seção 4).

Uso (num container, sem instalar nada no Windows):
  docker run --rm -v "$PWD:/w" -w /w python:3.12-alpine \
      python -I scripts/revisar_trilha.py pacote.json [escopo.json] [--sem-links]

pacote.json  o pacote de importação da trilha (POST /api/importar), com os documentos.
escopo.json  opcional: quais documentos estão no recorte e quais seções ficam de fora, com motivo:
             {"documentos": {"02-wifi": {"excluir": {"ancora": "motivo"}}}}

Imprime um relatório em Markdown e sai com 1 se algum critério falhar.
"""
import concurrent.futures
import json
import re
import sys
import urllib.error
import urllib.request

PALAVRAS_POR_MINUTO = 120   # texto técnico, lendo com atenção
MINUTOS_DE_TESTE = 10       # "Teste seu entendimento"
MINUTOS_DE_PRATICA = 5      # observar na prática (comandos, telas)
PALAVRAS_MINIMAS = 250      # abaixo disso a etapa é magra (rodada 1 da trilha piloto)

# Verbos por nível de Bloom (radicais, para pegar "explicar", "explique", "explicando").
VERBOS = {
    "lembrar": ["list", "nome", "nomea", "identific", "reconhec", "defin", "record", "localiz", "ach"],
    "entender": ["explic", "descrev", "distingu", "compar", "resum", "interpret", "classific", "diferenci", "relacion", "ilustr"],
    "aplicar": ["configur", "us", "execut", "calcul", "aplic", "demonstr", "mont", "ajust", "medi", "observ"],
    "analisar": ["diagnostic", "analis", "investig", "decomp", "examin", "rastre", "depur", "deduz", "previ", "prev"],
    "avaliar": ["avali", "justific", "decid", "escolh", "julg", "critic", "defend"],
    "criar": ["projet", "planej", "cri", "propo", "desenh", "elabor"],
}
ROTULOS = ("[oficial]", "[padrão]", "[livro]", "[curso]", "[artigo]", "[vídeo]")


def ancora(texto):
    """Âncora no formato do GitHub (o mesmo do app, frontend/src/markdown.js)."""
    t = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", texto)   # links: fica o texto
    t = re.sub(r"<[^>]*>", "", t).replace("`", "").replace("*", "")
    t = t.strip().lower()
    t = re.sub(r"[^\w\s-]", "", t)
    return re.sub(r"\s", "-", t)


def secoes(texto):
    """Lista de (nivel, ancora, titulo, inicio, fim) por título; fim = até o próximo de nível <= ."""
    linhas = texto.split("\n")
    titulos, usados, em_codigo = [], {}, False
    for i, linha in enumerate(linhas):
        if linha.startswith("```"):
            em_codigo = not em_codigo
        m = None if em_codigo else re.match(r"^(#{1,6})\s+(.*?)\s*#*\s*$", linha)
        if m:
            base = ancora(m.group(2))
            n = usados.get(base, 0)
            usados[base] = n + 1
            titulos.append([len(m.group(1)), base if n == 0 else f"{base}-{n}", m.group(2), i, len(linhas)])
    for k, t in enumerate(titulos):
        for prox in titulos[k + 1:]:
            if prox[0] <= t[0]:
                t[4] = prox[3]
                break
    return linhas, titulos


def palavras(texto):
    sem_codigo = re.sub(r"```.*?```", " ", texto, flags=re.S)
    return len(re.findall(r"\w+", sem_codigo))


def checar_link(url):
    pedido = urllib.request.Request(url, headers={"User-Agent": "trilha-de-estudo/revisor (homelab)"})
    try:
        with urllib.request.urlopen(pedido, timeout=20) as r:
            return url, r.status, ""
    except urllib.error.HTTPError as e:
        corpo = e.read(4000).decode("utf-8", "ignore").lower() if e.fp else ""
        anti_robo = e.code in (403, 429, 503) and any(p in corpo for p in ("anubis", "cloudflare", "captcha", "access denied", "bot"))
        return url, e.code, "anti-robô" if anti_robo else ""
    except Exception as e:  # rede, DNS, timeout
        return url, 0, str(e)[:80]


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    sem_links = "--sem-links" in sys.argv
    pacote = json.load(open(args[0], encoding="utf-8"))
    escopo = json.load(open(args[1], encoding="utf-8")) if len(args) > 1 else {"documentos": {}}

    docs = {d["slug"]: d["conteudo"] for d in pacote.get("documentos", [])}
    estrutura = {slug: secoes(texto) for slug, texto in docs.items()}
    t = pacote["trilha"]

    falhas, avisos = [], []
    falha = lambda c, msg: falhas.append(f"**{c}** {msg}")
    aviso = lambda c, msg: avisos.append(f"**{c}** {msg}")

    usados = {}          # doc -> set de âncoras citadas ("" = documento inteiro)
    links = {}           # url -> [etapas]
    linhas_tabela = []
    total = 0

    for m in t["modulos"]:
        n = len(m["etapas"])
        if not 2 <= n <= 6:
            falha("C4", f"módulo \"{m['titulo']}\" tem {n} etapa(s) (2 a 6)")
        for e in m["etapas"]:
            chave = e["chave"]
            minutos = e.get("minutos", 0)
            total += minutos
            # C2: verbo do nível declarado no objetivo
            bloom, obj = e.get("bloom", ""), e.get("objetivo", "").lower()
            if not obj or not bloom:
                falha("C2", f"`{chave}` sem objetivo ou sem nível de Bloom")
            elif not any(re.search(r"\b" + v, obj) for v in VERBOS[bloom]):
                falha("C2", f"`{chave}`: o objetivo não tem verbo de \"{bloom}\" ({obj[:70]}…)")
            # C3
            if not 20 <= minutos <= 90:
                falha("C3", f"`{chave}` com {minutos} min (20 a 90)")
            # C15
            if len(e.get("criterio", "").strip()) < 15:
                falha("C15", f"`{chave}` sem critério de conclusão verificável")
            # C9 e leitura medida
            mats = e.get("materiais", [])
            if not any(x.get("papel") == "principal" for x in mats) and not e.get("licao", "").strip():
                falha("C9", f"`{chave}` sem material principal nem lição")
            leitura = palavras(e.get("licao", ""))
            for x in mats:
                if x["tipo"] == "documento":
                    ref, anc = x["referencia"], x.get("localizador", "")
                    if ref not in docs:
                        falha("C10", f"`{chave}` cita o documento `{ref}`, que não está no pacote")
                        continue
                    linhas, tits = estrutura[ref]
                    usados.setdefault(ref, set()).add(anc)
                    if anc:
                        sec = next((s for s in tits if s[1] == anc), None)
                        if not sec:
                            falha("C10", f"`{chave}`: âncora `{ref}#{anc}` não existe")
                            continue
                        trecho = "\n".join(linhas[sec[3]:sec[4]])
                    else:
                        trecho = docs[ref]
                    if x.get("papel") == "principal":
                        leitura += palavras(trecho)
                elif x["tipo"] == "web":
                    links.setdefault(x["referencia"], []).append(chave)
                    if not any(r in x.get("nota", "") for r in ROTULOS):
                        falha("C12", f"`{chave}`: link \"{x['titulo']}\" sem rótulo de tipo na nota")
            estimado = round(leitura / PALAVRAS_POR_MINUTO) + MINUTOS_DE_TESTE + MINUTOS_DE_PRATICA
            if leitura and not (0.6 * estimado <= minutos <= 1.6 * estimado):
                falha("C14", f"`{chave}`: declarado {minutos} min, leitura medida pede ~{estimado} min (0,6× a 1,6×)")
            if leitura and leitura < PALAVRAS_MINIMAS:
                aviso("C3", f"`{chave}` é magra: {leitura} palavras de leitura principal; junte com a vizinha ou complete o material")
            linhas_tabela.append(f"| {m['titulo'][:28]} | `{chave}` | {bloom} | {minutos} | {leitura} | ~{estimado} |")
            # pré-requisitos (a ordem já é conferida pelo app; aqui só existência)

    # C8: seções em escopo não usadas
    for slug, conf in escopo.get("documentos", {}).items():
        if slug not in docs:
            falha("C8", f"documento em escopo `{slug}` não está no pacote")
            continue
        _, tits = estrutura[slug]
        cit = usados.get(slug, set())
        if "" in cit:
            continue
        excl = conf.get("excluir", {})
        for nivel, anc, titulo, ini, fim in tits:
            if nivel not in (2, 3):
                continue
            coberto = anc in cit or anc in excl
            # coberto por um ancestral citado ou excluído
            for n2, a2, _, i2, f2 in tits:
                if (a2 in cit or a2 in excl) and i2 <= ini and fim <= f2:
                    coberto = True
            # um h2 cujos filhos estão todos cobertos conta como coberto
            if not coberto and nivel == 2:
                filhos = [s for s in tits if s[0] == 3 and ini < s[3] < fim]
                coberto = bool(filhos) and all(
                    s[1] in cit or s[1] in excl for s in filhos
                )
            if not coberto:
                falha("C8", f"`{slug}#{anc}` ({titulo[:50]}) não é usada nem excluída")

    # C10 (documentos e lições): links internos entre documentos ("03-x.md#ancora") apontam para
    # âncoras que existem; C11 também testa os links externos citados nas lições.
    textos = dict(docs)
    for m in t["modulos"]:
        for e in m["etapas"]:
            if e.get("licao", "").strip():
                textos["lição " + e["chave"]] = e["licao"]
                for url in re.findall(r"https?://[^\s)>\]]+", e["licao"]):
                    links.setdefault(url.rstrip(".,;"), []).append(e["chave"] + " (lição)")
    for slug, texto in textos.items():
        for alvo, anc in re.findall(r"\]\((?:\.{1,2}/)*(?:[\w.-]+/)*([\w.-]+)\.md(?:#([^)\s]*))?\)", texto):
            if alvo not in docs:
                aviso("C10", f"`{slug}` liga para `{alvo}.md`, que não está no pacote")
            elif anc and not any(s[1] == anc for s in estrutura[alvo][1]):
                falha("C10", f"`{slug}` liga para `{alvo}#{anc}`, âncora que não existe")
        for anc in re.findall(r"\]\(#([^)\s]+)\)", texto):
            if not any(s[1] == anc for s in (estrutura[slug][1] if slug in estrutura else secoes(texto)[1])):
                falha("C10", f"`{slug}` liga para a própria âncora `#{anc}`, que não existe")

    # C11: links
    resultados = []
    if links and not sem_links:
        with concurrent.futures.ThreadPoolExecutor(8) as ex:
            resultados = list(ex.map(checar_link, links))
        for url, status, obs in resultados:
            if 200 <= status < 400:
                continue
            if obs == "anti-robô":
                aviso("C11", f"{url} → {status} (proteção anti-robô; não verificável aqui)")
            else:
                falha("C11", f"{url} → {status or 'sem resposta'} {obs} (em {', '.join(links[url])})")

    print(f"# Revisão automática: {t['titulo']}\n")
    print(f"- Etapas: {len(linhas_tabela)} · módulos: {len(t['modulos'])} · tempo declarado: {total} min ({total / 60:.1f} h)")
    print(f"- Links conferidos: {len(resultados)}{' (pulados)' if sem_links else ''}")
    print(f"- **Resultado: {'PASSOU' if not falhas else f'{len(falhas)} falha(s)'}**, {len(avisos)} aviso(s)\n")
    if falhas:
        print("## Falhas\n")
        print("\n".join(f"- {f}" for f in falhas) + "\n")
    if avisos:
        print("## Avisos\n")
        print("\n".join(f"- {a}" for a in avisos) + "\n")
    print("## Etapas\n")
    print("| Módulo | Etapa | Bloom | Min | Palavras | Leitura medida |")
    print("|---|---|---|---|---|---|")
    print("\n".join(linhas_tabela))
    sys.exit(1 if falhas else 0)


if __name__ == "__main__":
    main()
