# Build the React app, build the Go server, ship both in one small image.
FROM node:22 AS web
ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH
RUN corepack enable && corepack prepare pnpm@10.30.1 --activate
WORKDIR /app
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml frontend/.npmrc ./
RUN pnpm install --frozen-lockfile
COPY frontend/index.html frontend/vite.config.ts frontend/tsconfig.json frontend/components.json ./
COPY frontend/src ./src
# Vite bakes these into the bundle, so they are build arguments, not runtime
# variables.
ARG BYPASS_EMAIL_VERIFICATION=
ARG TURNSTILE_SITE_KEY=
ARG POSTHOG_PUBLIC_KEY=
ARG POSTHOG_HOST=
RUN BYPASS_EMAIL_VERIFICATION=$BYPASS_EMAIL_VERIFICATION     TURNSTILE_SITE_KEY=$TURNSTILE_SITE_KEY POSTHOG_PUBLIC_KEY=$POSTHOG_PUBLIC_KEY     POSTHOG_HOST=$POSTHOG_HOST pnpm exec vite build

FROM golang:1.27.3 AS server
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/erase-user ./cmd/erase-user

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/server /out/erase-user /
# The image runs as nonroot (65532). Files copied from the build context
# (frontend/src/public: logo, favicons) keep the uploader's mode, which can be
# owner-only, and Go then answers 403 for them. Owning /dist fixes that.
COPY --from=web --chown=65532:65532 /app/dist /dist
ENV STATIC_DIR=/dist
EXPOSE 8080
ENTRYPOINT ["/server"]
