# Beta Release Sonrası Kontrol Listesi

## 1. GitHub Actions İzleme (5-10 dk)

🔗 **Kontrol linki:** https://github.com/ersinkoc/karadul/actions

Workflow'ların tamamlanmasını bekleyin:
- [ ] CI workflow (test + build)
- [ ] Release workflow (binary oluşturma)
- [ ] Container workflow (Docker image)

## 2. Release Doğrulama

GitHub Releases sayfasında kontrol edin:
🔗 https://github.com/ersinkoc/karadul/releases/tag/v0.1.0-beta.1

Kontrol listesi:
- [ ] 10 binary mevcut (linux-amd64, darwin-arm64, windows-amd64.exe, vb.)
- [ ] checksums.txt dosyası var
- [ ] Release notları doğru
- [ ] "Pre-release" olarak işaretli

## 3. Docker Image Kontrolü

🔗 https://github.com/ersinkoc/karadul/pkgs/container/karadul

```bash
# Test et
docker pull ghcr.io/ersinkoc/karadul:v0.1.0-beta.1
docker run --rm ghcr.io/ersinkoc/karadul:v0.1.0-beta.1 version
```

## 4. Homebrew Tap Kurulumu (Opsiyonel)

Homebrew desteği için ayrı repo oluşturun:

```bash
# Yeni repo oluştur (GitHub web arayüzünden veya gh CLI)
gh repo create homebrew-karadul --public --description "Homebrew tap for Karadul"

# Clone et
git clone https://github.com/ersinkoc/homebrew-karadul.git
cd homebrew-karadul

# Formula oluştur
mkdir -p Formula
cp ../karadul/contrib/homebrew/karadul.rb.template Formula/karadul.rb

# SHA256 hash'leri güncelle (release'deki checksums.txt'den al)
# Edit: Formula/karadul.rb

git add Formula/karadul.rb
git commit -m "karadul: add v0.1.0-beta.1"
git push
```

## 5. Beta Test Daveti

GitHub Discussions veya Issues'da beta test duyurusu:

```markdown
## 🧪 Karadul v0.1.0-beta.1 Yayında!

Bu beta sürümü test etmenizi ve geri bildirimde bulunmanızı rica ediyoruz.

### Yeni Özellikler
- Windows desteği (Wintun)
- Firewall yönetimi
- Docker desteği

### Test Edilecekler
- [ ] Windows kurulumu (wintun.dll)
- [ ] `karadul wintun-check` komutu
- [ ] `karadul firewall setup` komutu
- [ ] Basic mesh bağlantısı

### Bildirim
Sorunları [GitHub Issues](../../issues) üzerinden bildirin.

**Not:** Bu bir beta sürümüdür. Production kullanımı için v0.1.0 stable sürümünü bekleyin.
```

## 6. Sonraki Adımlar (v0.1.0 Stable)

Beta feedback toplandıktan sonra:

1. Windows test raporları
2. Hata düzeltmeleri
3. Dokümantasyon güncellemeleri
4. `v0.1.0` stable release

## Yardım ve Destek

Sorularınız için:
- GitHub Discussions: https://github.com/ersinkoc/karadul/discussions
- GitHub Issues: https://github.com/ersinkoc/karadul/issues
