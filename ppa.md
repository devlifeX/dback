# راهنمای انتشار DBack در Launchpad PPA

این سند برای publish بعدی DBack در PPA است. با یک دستور build، sign و upload انجام می‌شود.

| مورد | مقدار |
|------|--------|
| PPA | `ppa:devlifex/dback` |
| صفحه PPA | https://launchpad.net/~devlifex/+archive/ubuntu/dback |
| Maintainer | `Dariush Vesal <dvworkmail2017@gmail.com>` |
| GPG Key ID | `E00C906928B7599C` |
| GPG Fingerprint | `E8B2 5AD6 8688 EC02 4359 E03F E00C 9069 28B7 599C` |
| اسکریپت | `./packaging/build-ppa.sh` |
| بسته‌بندی Debian | `debian/` |

---

## یک‌بار — نصب ابزارها

```bash
sudo apt install devscripts debhelper build-essential \
  pkg-config libvulkan-dev xorg-dev libwayland-dev \
  libxkbcommon-dev libxkbcommon-x11-dev libx11-xcb-dev \
  libxcursor-dev libxfixes-dev libegl-dev
```

---

## یک‌بار — GPG و Launchpad

### کلید GPG کجاست؟

```
~/.gnupg/
├── pubring.kbx              # کلید عمومی
├── private-keys-v1.d/       # کلید خصوصی (backup بگیر!)
├── gpg.conf                 # default-key
└── trustdb.gpg
```

### Backup کلید (مهم)

```bash
gpg --armor --export-secret-keys E00C906928B7599C > ~/dback-gpg-private-backup.asc
gpg --armor --export E00C906928B7599C > ~/dback-gpg-public.asc
```

### تنظیم debuild

```bash
export DEBFULLNAME="Dariush Vesal"
export DEBEMAIL="dvworkmail2017@gmail.com"
grep -q 'default-key E00C906928B7599C' ~/.gnupg/gpg.conf 2>/dev/null || \
  echo "default-key E00C906928B7599C" >> ~/.gnupg/gpg.conf
```

این خطوط را به `~/.bashrc` هم اضافه کن.

### ثبت کلید در Launchpad

1. برو به: https://launchpad.net/~devlifex/+editpgpkeys
2. Fingerprint را وارد کن:
   ```
   E8B2 5AD6 8688 EC02 4359 E03F E00C 9069 28B7 599C
   ```
3. ایمیل `dvworkmail2017@gmail.com` را تأیید کن تا کلید **Active** شود.

اگر keyserver خطا داد:

```bash
gpg --keyserver hkp://keyserver.ubuntu.com:80 --send-keys E00C906928B7599C
# یا کلید عمومی را مستقیم paste کن:
gpg --armor --export E00C906928B7599C
```

### ساخت PPA (اگر وجود ندارد)

https://launchpad.net/~/+archive/ubuntu/ppa/new — نام: `dback`

Ubuntu series پیشنهادی: **22.04 jammy**, **24.04 noble**

---

## هر release — چک‌لیست نسخه

قبل از upload، این فایل‌ها باید **هم‌نسخه** باشند:

| فایل | فیلد |
|------|------|
| `main.go` | `appVersion` |
| `build.sh` | `APP_VERSION` |
| `debian/changelog` | خط اول، مثلاً `dback (3.6.4)` |

### به‌روز کردن changelog

release جدید upstream:

```bash
dch -v 3.6.4 "New upstream release."
```

فقط fix بسته‌بندی (همان upstream):

```bash
dch -i
# نتیجه: 3.6.3-1 ، 3.6.3-2 ، ...
```

> بسته **native** است؛ برای اولین upload هر upstream از `3.6.4` (بدون `-1`) استفاده کن.

---

## publish — دستور اصلی

```bash
cd ~/dev/backend/dback
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

این کارها را انجام می‌دهد:

1. ساخت source package (`debuild -S -sa -d -us -uc`)
2. sign با fingerprint کلید GPG (`debsign -k …`)
3. upload به Launchpad (`dput ppa:devlifex/dback …`)

خروجی در **پوشه parent** ریپو:

```
../dback_3.6.3.dsc
../dback_3.6.3.tar.xz
../dback_3.6.3_source.changes
```

### فقط build + sign (بدون upload)

```bash
./packaging/build-ppa.sh
dput ppa:devlifex/dback ../dback_3.6.3_source.changes
```

### upload دستی بعداً

```bash
debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ../dback_3.6.3_source.changes
dput ppa:devlifex/dback ../dback_3.6.3_source.changes
```

---

## بعد از upload

1. برو به: https://launchpad.net/~devlifex/+archive/ubuntu/dback
2. منتظر بمان تا buildها **Published** شوند (۱۰–۳۰ دقیقه)
3. تست نصب:

```bash
sudo add-apt-repository ppa:devlifex/dback
sudo apt update
sudo apt install dback
dback
```

---

## دستور نصب برای کاربران (README / سایت)

```bash
sudo add-apt-repository ppa:devlifex/dback
sudo apt update
sudo apt install dback
```

---

## خطاهای رایج

### `No secret key` هنگام sign

علت: `debuild` با نام کامل maintainer sign می‌کند و GPG key را پیدا نمی‌کند.

راه‌حل: از اسکریپت `./packaging/build-ppa.sh` استفاده کن (خودش با key ID/fingerprint sign می‌کند). یا:

```bash
export GPG_TTY=$(tty)
debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ../dback_3.6.3_source.changes
```

### `Connection reset by peer` در dput

FTP Launchpad گاهی قطع می‌شود. **دوباره همان دستور `dput` را بزن** — معمولاً بار دوم موفق می‌شود. اسکریپت `UPLOAD=1` خودش تا ۳ بار retry می‌کند.

### lintian: `bad-distribution-in-changes-file ubuntu`

برای PPA مشکلی نیست؛ upload قبول می‌شود.

### lintian: `build-depends-on-build-essential` / `xorg-dev`

هشدار policy است؛ مانع upload یا build در Launchpad نمی‌شود.

### ایمیل GPG با changelog فرق دارد

ایمیل کلید GPG **باید** با `debian/changelog` یکی باشد: `dvworkmail2017@gmail.com`

### keyserver خطا می‌دهد

```bash
gpg --keyserver hkp://keyserver.ubuntu.com:80 --send-keys E00C906928B7599C
```

یا کلید armored را مستقیم در Launchpad paste کن.

---

## ساختار بسته Debian

| مسیر نصب | فایل |
|----------|------|
| `/usr/lib/dback/dback` | باینری |
| `/usr/bin/dback` | launcher |
| `/usr/share/applications/dback.desktop` | منوی برنامه |
| `/usr/share/icons/hicolor/*/apps/dback.png` | آیکون |

Build در Launchpad: Go ≥ 1.25 (در صورت نیاز `debian/prepare-go.sh` دانلود می‌کند).

---

## GitHub Actions — آپلود خودکار PPA

با push کردن tag (`v*`)، workflow [`.github/workflows/ppa.yml`](.github/workflows/ppa.yml) خودکار source package را sign و به PPA می‌فرستد (موازی با GitHub Release).

### یک‌بار — Secrets در GitHub

دو روش — **یکی** کافی است:

#### روش A — Repository secrets (ساده‌تر)

Repo → **Settings → Secrets and variables → Actions** → **Repository secrets** → **New repository secret**

| Secret | نام دقیق | مقدار |
|--------|----------|--------|
| کلید خصوصی | **`PPA_GPG_PRIVATE_KEY`** | کل فایل armored |
| passphrase | **`PPA_GPG_PASSPHRASE`** | رمز کلید؛ اگر ندارد یک فاصله ` ` |

اگر از این روش استفاده می‌کنی، خط `environment:` را از [`.github/workflows/ppa.yml`](.github/workflows/ppa.yml) حذف کن.

#### روش B — Environment secrets (همان setup فعلی تو)

Secrets را زیر **Environment** به نام **`PPA_GPG_PASSPHRASE`** گذاشتی — workflow الان با این environment کار می‌کند:

```yaml
environment: PPA_GPG_PASSPHRASE
```

| Secret | Environment |
|--------|-------------|
| `PPA_GPG_PRIVATE_KEY` | `PPA_GPG_PASSPHRASE` |
| `PPA_GPG_PASSPHRASE` | `PPA_GPG_PASSPHRASE` |

> **Repository secrets** و **Environment secrets** جدا هستند. اگر فقط Environment پر باشد و workflow `environment:` نداشته باشد، secret خالی دیده می‌شود.

استخراج کلید خصوصی:

```bash
gpg --armor --export-secret-keys E00C906928B7599C > ppa-private-key.asc
cat ppa-private-key.asc
```

**همه** خروجی `cat` را paste کن. **هرگز** commit نکن.

### خطا: `PPA_GPG_PRIVATE_KEY is missing`

- secret در **Repository** است ولی workflow `environment:` دارد → یا environment را پر کن یا `environment:` را حذف کن
- secret فقط در **Environment** است ولی workflow `environment:` ندارد → `environment: PPA_GPG_PASSPHRASE` اضافه کن (الان fix شده)
- نام secret دقیقاً `PPA_GPG_PRIVATE_KEY` باشد
- بعد از fix، workflow را **Re-run** کن

> توصیه امنیتی: برای CI می‌توانی یک **subkey** فقط-sign بسازی؛ فعلاً همان کلید اصلی هم کار می‌کند.

### release با CI

```bash
# 1. نسخه را در main.go و build.sh هماهنگ کن
# 2. debian/changelog را به‌روز کن (CI خودش هم sync می‌کند اگر فرق داشت)
git add main.go build.sh debian/changelog
git commit -m "Release 3.6.4"
git tag v3.6.4
git push origin master v3.6.4
```

بعد از push tag:
- `go.yml` → GitHub Release (linux, windows, .deb)
- `ppa.yml` → Launchpad PPA upload

وضعیت Actions: https://github.com/devlifeX/dback/actions

---

## release بعدی — مثال کامل (دستی)

```bash
# 1. نسخه را در main.go و build.sh به 3.6.4 تغییر بده
dch -v 3.6.4 "New upstream release."

# 2. commit + tag (اختیاری، برای GitHub Release)
git add main.go build.sh debian/changelog
git commit -m "Release 3.6.4"
git tag v3.6.4
git push origin master v3.6.4

# 3. PPA upload
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

---

## فایل‌های مرتبط

| فایل | نقش |
|------|------|
| `ppa.md` | این راهنما |
| `packaging/build-ppa.sh` | build + sign + upload |
| `debian/` | بسته منبع Launchpad |
| `packaging/sync-debian-changelog.sh` | هماهنگ‌سازی changelog با tag (CI) |
| `.github/workflows/ppa.yml` | آپلود خودکار PPA روی tag |
