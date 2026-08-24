FROM node:20-alpine

WORKDIR /app

# The repo ships a pnpm lockfile, so install with pnpm to honour it.
RUN corepack enable
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install

COPY . .

EXPOSE 5173
CMD ["pnpm", "run", "dev"]
