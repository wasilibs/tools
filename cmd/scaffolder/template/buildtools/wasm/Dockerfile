FROM ghcr.io/webassembly/wasi-sdk:wasi-sdk-21

RUN apt-get update && apt-get install -y curl binaryen

ADD buildtools/wasm/version.txt version.txt
