.PHONY: dev-api dev-web build-web test test-go test-web deploy

dev-api:
	cd backend && DATA_DIR=../data go run .

dev-web:
	cd frontend && npm run dev

# O binário Go embute o PWA; para rodar sem o Vite, gere o build dentro do backend.
build-web:
	cd frontend && npm run build && rm -rf ../backend/web/dist/* && cp -R dist/. ../backend/web/dist/

test: test-go test-web

test-go:
	cd backend && go test ./...

test-web:
	cd frontend && npm test

deploy:
	./deploy.sh
