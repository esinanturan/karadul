## 🧪 Karadul v0.1.0-beta.1 Beta Duyurusu

Karadul mesh VPN'in ilk beta sürümü yayında!

### 📦 İndirme

**GitHub Release:** https://github.com/ersinkoc/karadul/releases/tag/v0.1.0-beta.1

**Platformlar:**
- Linux (amd64, arm64, armv7)
- macOS (Intel & Apple Silicon)
- Windows (amd64, arm64, x86) ⚠️ Beta
- FreeBSD/OpenBSD (amd64)

**Docker:**
```bash
docker pull ghcr.io/ersinkoc/karadul:v0.1.0-beta.1
```

### ✨ Yeni Özellikler

1. **Windows Desteği (Beta)**
   - WireGuard Wintun driver entegrasyonu
   - `karadul wintun-check` komutu
   - Windows Firewall yönetimi

2. **Güvenlik Duvarı Yönetimi**
   - `karadul firewall setup` - Otomatik kurallar
   - `karadul firewall check` - Durum kontrolü
   - `karadul firewall allow-port` - Port izinleri

3. **Kurulum Seçenekleri**
   - Binary (doğrudan indirme)
   - Docker image
   - Homebrew (yakında)

### ⚠️ Önemli Notlar

**Windows Kullanıcıları:**
Wintun DLL'sini manuel olarak indirmeniz gerekir:
```powershell
# PowerShell ile indir
Invoke-WebRequest -Uri "https://www.wintun.net/builds/wintun-0.14.1-amd64.zip" -OutFile "wintun.zip"
Expand-Archive wintun.zip
Copy-Item wintun\wintun.dll C:\Windows\System32\
```

Veya `karadul.exe` ile aynı dizine koyun.

**Kontrol:**
```bash
karadul wintun-check
```

### 🧪 Test Edilecekler

- [ ] Temel mesh bağlantısı (Linux/macOS)
- [ ] Windows TUN oluşturma
- [ ] Windows firewall kuralları
- [ ] Docker container çalıştırma
- [ ] NAT traversal (farklı ağlar)

### 📝 Geri Bildirim

Sorun ve önerileriniz için:
- 🐛 [GitHub Issues](https://github.com/ersinkoc/karadul/issues)
- 💬 [GitHub Discussions](https://github.com/ersinkoc/karadul/discussions)

### 🔒 Güvenlik

Beta sürüm production kullanımı için önerilmez. Test ortamında kullanın.

---

**Tam sürüm (v0.1.0) ne zaman?**
Beta testleri tamamlandıktan ve Windows desteği stabilize edildikten sonra stable sürüm yayınlanacak.

**Star atmayı unutmayın!** ⭐
https://github.com/ersinkoc/karadul
