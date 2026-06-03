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
| `debian/changelog` | خط اول، مثلاً `dback (3.6.8-1~ubuntu24.04.1) noble` |

### به‌روز کردن changelog

**distribution** باید codename Ubuntu باشد (`noble`, `jammy`) — **نه** `ubuntu`.

release جدید برای noble (24.04):

```bash
dch -v 3.6.5-1 -D noble "Release 3.6.5."
```

fix بسته‌بندی:

```bash
dch -i -D noble
```

> Launchpad فقط **source package** می‌پذیرد (`debuild -S`)، نه فایل `.deb` آماده.

---

## publish — دستور اصلی

```bash
cd ~/dev/backend/dback
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

> **مهم:** اسکریپت را بدون `| tail` اجرا کن — در غیر این صورت خروجی و promptهای GPG دیده نمی‌شوند و build «گیر کرده» به نظر می‌رسد.

این کارها را انجام می‌دهد:

1. ساخت source package (`debuild -S -sa -d -us -uc`)
2. sign با fingerprint کلید GPG (`debsign -k …`)
3. upload به Launchpad (`dput ppa:devlifex/dback …`)

خروجی در **پوشه parent** ریپو:

```
../dback_3.6.4-1.dsc
../dback_3.6.4-1.tar.xz
../dback_3.6.4-1_source.changes
```

### فقط build + sign (بدون upload)

```bash
./packaging/build-ppa.sh
dput ppa:devlifex/dback ../dback_3.6.4-1_source.changes
```

### upload دستی بعداً

```bash
debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ../dback_3.6.4-1_source.changes
dput -f ppa:devlifex/dback ../dback_3.6.4-1_source.changes
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

## طبق مستندات Launchpad (خلاصه)

منبع: [Upload a package to a PPA](https://help.launchpad.net/Packaging/PPA/Uploading)

### روش upload (FTP — همان کاری که ما می‌کنیم)

```bash
dput ppa:devlifex/dback ../dback_VERSION_source.changes
```

- Launchpad خودش **build** می‌گیرد؛ فایل `.deb` آماده upload **نمی‌شود**
- source با `debuild -S` ساخته می‌شود (`./packaging/build-ppa.sh`)

### `debian/changelog` — فیلد distribution

Suite در changelog باید یکی از **Ubuntu series فعال در PPA** باشد:

| Ubuntu | codename در changelog |
|--------|------------------------|
| 24.04 | `noble` |
| 22.04 | `jammy` |

❌ `ubuntu` — معتبر نیست → build شروع نمی‌شود

### چند Ubuntu series (jammy + noble)

Launchpad **نسخه Debian یکتا برای هر series** می‌خواهد. CI روی tag `v*` دو job دارد (matrix در `.github/workflows/ppa.yml`).

| Series | distribution | نمونه version |
|--------|--------------|----------------|
| noble (24.04) | `noble` | `3.6.8-1~ubuntu24.04.1` |
| jammy (22.04) | `jammy` | `3.6.8-1~ubuntu22.04.1` |

[`packaging/sync-debian-changelog.sh`](packaging/sync-debian-changelog.sh) با `PPA_DIST=noble|jammy` changelog را تنظیم می‌کند.

**Go روی jammy:** Build-Depends از `golang-1.22-go` استفاده می‌کند (پیش‌فرض jammy فقط 1.18 دارد). [`debian/prepare-go.sh`](debian/prepare-go.sh) باینری Go را از PATH یا `/usr/lib/go-1.22/bin/go` پیدا می‌کند.

**پیش‌نیاز Launchpad:** در تنظیمات PPA هر دو series **jammy** و **noble** فعال باشند.

```bash
# noble
PPA_DIST=noble APP_VERSION=3.6.8 ./packaging/sync-debian-changelog.sh
UPLOAD=1 ./packaging/build-ppa.sh

# jammy (نسخه متفاوت!)
PPA_DIST=jammy APP_VERSION=3.6.8 ./packaging/sync-debian-changelog.sh
UPLOAD=1 ./packaging/build-ppa.sh
```

### dput پیشرفته (اختیاری)

فایل نمونه: [`packaging/dput.cf.example`](packaging/dput.cf.example)

```ini
[dback-ppa]
incoming = ~devlifex/ubuntu/dback/
```

```bash
dput dback-ppa ../dback_3.6.4-1_source.changes
```

### SFTP (جایگزین FTP)

در `dput.cf`: `method = sftp` و `login = devlifex` — SSH fingerprint را تأیید کن.

---

## خطاهای رایج

### `No secret key` هنگام sign

علت: `debuild` با نام کامل maintainer sign می‌کند و GPG key را پیدا نمی‌کند.

راه‌حل: از اسکریپت `./packaging/build-ppa.sh` استفاده کن (خودش با key ID/fingerprint sign می‌کند). یا:

```bash
export GPG_TTY=$(tty)
debsign -k E8B25AD68688EC024359E03FE00C906928B7599C ../dback_3.6.3_source.changes
```

### GPG verification failed on .dsc (`No data`)

```
GPG verification of ... dback_X.dsc failed: (7, 58, 'No data')
```

علت: فایل `.dsc` **بدون امضای معتبر** upload شده — معمولاً در CI وقتی `debsign` بعد از build درست کار نمی‌کند.

**راه‌حل:** اسکریپت الان با `debuild -k FINGERPRINT` یک‌جا sign می‌کند و قبل از upload `gpg --verify` روی `.dsc` و `.changes` می‌زند.

```bash
rm -f ../dback_*
export GPG_TTY=$(tty)
UPLOAD=1 ./packaging/build-ppa.sh
```

اگر `GPG signature verification passed` دیدی، upload کن. نسخه جدید: `3.6.5-2`.

### MD5 mismatch (Launchpad rejection)

```
File dback_X.tar.xz mentioned in the changes has a MD5 mismatch
```

علت: فایل `.changes` / `.dsc` با `tar.xz` واقعی که Launchpad خوانده **هم‌خوان نیست** — معمولاً وقتی چند build همزمان artifactهای هم‌نام در `../dback_*` را overwrite کرده‌اند. **کلید GPG فعال بودن کافی نیست**؛ این خطا از قاطی شدن فایل‌هاست.

**راه‌حل:**

1. revision جدید بزن (مثلاً `3.6.5-3`، نه retry همان `3.6.5-2`):

```bash
dch -i -D noble
```

2. فقط از اسکریپت با lock و staging upload کن (همزمان دو terminal اجرا نکن):

```bash
cd ~/dev/backend/dback
export GPG_TTY=$(tty)
SIGN=1 UPLOAD=1 ./packaging/build-ppa.sh
```

اسکریپت قبل از `dput` checksum و GPG را verify می‌کند، سپس یک **کپی ایزوله** در `.dback-upload-stage.*` می‌سازد و فقط همان را upload می‌کند.

### `Connection reset by peer` در dput

FTP Launchpad گاهی قطع می‌شود. **دوباره همان دستور `dput` را بزن** — معمولاً بار دوم موفق می‌شود. اسکریپت `UPLOAD=1` خودش تا ۳ بار retry می‌کند.

### lintian: `bad-distribution-in-changes-file ubuntu`

**مهم:** در `debian/changelog` به‌جای `ubuntu` باید codename سری Ubuntu باشد، مثلاً **`noble`** (24.04) یا **`jammy`** (22.04).  
PPA فقط برای seriesهایی که در تنظیمات PPA فعال کرده‌ای build می‌گیرد.

```bash
dch -v 3.6.4-1 -D noble "Fix PPA target series."
```

### Builds خالی / «No matching builds»

چک‌لیست:

1. **ایمیل** `dvworkmail2017@gmail.com` — Launchpad acceptance یا rejection
2. **PPA فعال است؟** https://launchpad.net/~devlifex/+activate-ppa
3. **Ubuntu series فعال؟** https://launchpad.net/~devlifex/+archive/ubuntu/dback/+edit → حداقل **noble** و/یا **jammy**
4. **کلید GPG Active** (نه pending): https://launchpad.net/~devlifex/+editpgpkeys
5. **changelog** distribution = `noble` نه `ubuntu`
6. **نسخه تکراری** — Launchpad نسخه تکراری قبول نمی‌کند؛ revision را بالا ببر (`3.6.4-1`) و `dput -f` بزن

```bash
rm -f ../dback_*_source.ppa.upload
UPLOAD=1 ./packaging/build-ppa.sh
```

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

Build در Launchpad: از **Go سیستم** نوبل (1.22) و **`vendor/` داخل source tarball** استفاده می‌شود — بدون اینترنت روی builder (`GOPROXY=off`, `GOTOOLCHAIN=local`). پوشه `vendor/` در git نیست؛ `./packaging/build-ppa.sh` قبل از `debuild` با `go mod vendor` می‌سازد (نیاز به شبکه فقط روی ماشین build/upload).

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
git commit -m "Release 3.6.5"
git tag v3.6.5
git push origin master v3.6.5
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
git commit -m "Release 3.6.5"
git tag v3.6.5
git push origin master v3.6.5

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
