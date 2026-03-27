# Karadul Web UI

Modern React 19 + TypeScript + Vite + Tailwind CSS + shadcn/ui ile geliştirilmiş Web arayüzü.

## Özellikler

- **Dashboard**: Sistem metrikleri, node ve peer istatistikleri
- **Topology**: React Flow ile mesh network görselleştirmesi
- **Nodes**: Node yönetimi (listeleme, detay, silme)
- **Peers**: Peer bağlantıları ve durumları
- **Settings**: Auth key ve ACL yönetimi
- **Real-time**: WebSocket üzerinden canlı güncellemeler
- **Dark/Light Mode**: Tema desteği

## Teknoloji Stack

- React 19
- TypeScript 5.6
- Vite 5.4
- Tailwind CSS 3.4
- shadcn/ui (Radix UI)
- React Query (TanStack Query)
- Zustand (State Management)
- React Flow (Topology Graph)
- Recharts (Charts)
- WebSocket

## Geliştirme

```bash
# Bağımlılıkları yükle
npm install

# Geliştirme sunucusu (port 5173)
npm run dev

# Derleme
npm run build

# Lint
npm run lint

# Preview (build sonrası)
npm run preview
```

## API Entegrasyonu

Vite proxy yapılandırması `vite.config.ts` içinde:
- `/api` → `http://localhost:8080`
- `/ws` → `ws://localhost:8080`

## Ortam Değişkenleri

`.env` dosyası oluşturun:

```env
# API temel URL
VITE_API_BASE_URL=/api

# WebSocket URL
VITE_WS_URL=ws://localhost:8080/ws

# Mock API (sadece geliştirme için)
VITE_USE_MOCK_API=false
```

## Build ve Go Embed

Production build:

```bash
cd web
npm run build
```

Derlenen dosyalar `web/dist/` dizinine çıkar. Go backend bu dosyaları embed eder.

## Dizin Yapısı

```
web/
├── src/
│   ├── components/
│   │   ├── ui/          # shadcn/ui bileşenleri
│   │   ├── layout.tsx   # Ana layout
│   │   ├── sidebar.tsx  # Navigasyon
│   │   └── header.tsx   # Üst bar
│   ├── lib/
│   │   ├── api.ts       # API hooks (React Query)
│   │   ├── store.ts     # Zustand store
│   │   ├── websocket.tsx # WebSocket provider
│   │   └── utils.ts     # Yardımcı fonksiyonlar
│   ├── pages/
│   │   ├── dashboard.tsx
│   │   ├── topology.tsx
│   │   ├── nodes.tsx
│   │   ├── peers.tsx
│   │   └── settings.tsx
│   ├── App.tsx
│   └── main.tsx
├── index.html
├── package.json
├── tailwind.config.js
├── tsconfig.json
└── vite.config.ts
```

## Lisans

MIT License - Karadul projesi ile aynı.
