---
description: Commit perubahan kode ke Git dengan conventional commit message
---

// turbo-all

# Git Commit

Workflow untuk commit perubahan kode ke repository Git menggunakan format **Conventional Commits**.

## Format Commit Message

```
<tipe>(<scope>): <deskripsi singkat>
```

### Tipe yang Tersedia

| Tipe       | Keterangan                                            |
| ---------- | ----------------------------------------------------- |
| `feat`     | Fitur baru                                            |
| `fix`      | Perbaikan bug                                         |
| `docs`     | Perubahan dokumentasi                                 |
| `style`    | Formatting, titik koma, dll (bukan perubahan kode)    |
| `refactor` | Refactoring kode tanpa tambah fitur atau perbaiki bug |
| `test`     | Menambah atau memperbaiki tes                         |
| `chore`    | Perubahan build, tools, dependensi, dll               |
| `perf`     | Peningkatan performa                                  |
| `ci`       | Perubahan CI/CD                                       |

### Contoh Scope (Opsional)

message: pake bahasa inggris

`auth`, `user`, `db`, `middleware`, `config`, `docs`

## Langkah-langkah

1. Cek status perubahan:

```bash
git status --short
```

2. Cek diff perubahan:

```bash
git diff --stat
```

3. **Wajib Commit Per File** (Jangan pakai `git add .`):

Untuk SETIAP file yang berubah, lakukan commit satu per satu agar message-nya spesifik sesuai perubahan di file tersebut.

```bash
git add <path/ke/file>
git commit -m "<tipe>(<scope>): <deskripsi spesifik untuk file ini>"
```

### Contoh Commit Per File

```bash
git add internal/handler/user.handler.go
git commit -m "feat(user): tambah endpoint update role"

git add internal/repository/user.repository.go
git commit -m "feat(user): tambah query update role di repository"
```

### Contoh Format Commit Message

```bash
# Fitur baru
git commit -m "feat(product): tambah endpoint CRUD produk"

# Perbaikan bug
git commit -m "fix(auth): perbaiki validasi token expired"

# Dokumentasi
git commit -m "docs(swagger): update anotasi endpoint user"

# Refactoring
git commit -m "refactor(repository): pindah query builder ke common helper"

# Migrasi database
git commit -m "chore(db): tambah migrasi tabel orders"
```

4. (Opsional) Push ke remote:

```bash
git push origin <nama-branch>
```
