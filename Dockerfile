# Stage 1 - Build Frontend with esbuild-wasm for Linux(Debian) & Windows
FROM node:18 AS frontend-builder

WORKDIR /app
COPY frontend/package*.json ./frontend/
RUN sed -i 's/"esbuild"/"esbuild-wasm"/g' frontend/package.json
RUN cd frontend && npm install --include=dev
COPY frontend ./frontend/
RUN cd frontend && npm run build

# Stage 2 - Build Backend for Linux (Linux Builder)
FROM --platform=linux/amd64 golang:1.21 AS linux-builder

# Install dependencies for Linux
RUN apt-get update && apt-get install -y \
    gcc \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev 

WORKDIR /app
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY . .
RUN wails build -platform linux/amd64 -clean -o election-linux

# Stage 3 - Build Backend for Windows (Windows Builder)
FROM --platform=windows/amd64 golang:1.21 AS windows-builder

# Install dependencies for Windows cross-compile
RUN apt-get update && apt-get install -y \
    gcc-mingw-w64-x86-64

WORKDIR /app
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY . .
RUN GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc wails build -platform windows/amd64 -clean -o election-windows.exe

# Stage 4 - Export Linux
FROM scratch AS export-linux
COPY --from=linux-builder /app/build/bin/election-linux /

# Stage 5 - Export Windows
FROM scratch AS export-windows
COPY --from=windows-builder /app/build/bin/election-windows.exe /
