---
description: Build binary production untuk Web Profile Backend
---

# Build Binary Production

## Langkah-langkah

// turbo

1. Build binary Go:

```bash
go build -o web-profile-backend.exe .
```

// turbo 2. Pastikan binary sudah terbuat:

```bash
ls -la web-profile-backend.exe
```

3. (Opsional) Jalankan binary production:

```bash
./web-profile-backend.exe
```

> **Catatan:** Pastikan file `.env` sudah dikonfigurasi dengan benar sebelum menjalankan binary.
