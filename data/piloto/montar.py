"""Monta o pacote de importação: estrutura (JSON) + documentos de estudo/ + lições (.md).

Uso: python -I montar.py estrutura-vN.json pacote-vN.json
Os documentos vêm de /docs (estudo/ e casa-inteligente-rede.md, montados só para leitura).
Cada etapa pode ter "licao_arquivo": o nome de um .md em /w/licoes, que vira o campo "licao".
"""
import glob
import json
import os
import sys

estrutura = json.load(open(sys.argv[1], encoding="utf-8"))
documentos = []
for f in sorted(glob.glob("/docs/estudo/*.md")) + ["/docs/casa-inteligente-rede.md"]:
    slug = os.path.basename(f)[:-3]
    documentos.append({"slug": slug, "conteudo": open(f, encoding="utf-8").read()})

for m in estrutura["modulos"]:
    for e in m["etapas"]:
        arq = e.pop("licao_arquivo", None)
        if arq:
            e["licao"] = open(os.path.join("/w/licoes", arq), encoding="utf-8").read()

pacote = {"substituir": True, "documentos": documentos, "trilha": estrutura}
json.dump(pacote, open(sys.argv[2], "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print(f"{sys.argv[2]}: {len(documentos)} documentos, {sum(len(m['etapas']) for m in estrutura['modulos'])} etapas")
