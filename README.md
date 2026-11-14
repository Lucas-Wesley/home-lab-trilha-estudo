# Trilha de Estudo

Trilhas de estudo profundas, personalizadas e em ordem, com IA que pergunta, propõe e aceita
comentários. Parte do ecossistema de apps pessoais do homelab.

- Plano e decisões: [`docs/DECISOES.md`](docs/DECISOES.md)
- Pedagogia: [`docs/PEDAGOGIA.md`](docs/PEDAGOGIA.md)
- Design system: [`docs/DESIGN-SYSTEM.md`](docs/DESIGN-SYSTEM.md)

## Desenvolvimento

Requisitos: Go 1.24+ e Node 22+ (ou só Docker).

```sh
make dev-api   # API em http://localhost:8080 (dados em ./data)
make dev-web   # front em http://localhost:5173, com proxy de /api
make test      # testes do Go e do front
```

Imagem de produção local: `docker compose up --build` (porta `127.0.0.1:8080`).

## Publicar

`./deploy.sh` publica o commit atual de `origin/main` na VM `apps` do homelab.
