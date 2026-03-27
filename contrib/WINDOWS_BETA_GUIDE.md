# Windows Beta Kullanım Kılavuzu

Bu kılavuz Karadul v0.1.0-beta.1 Windows kurulumu için hazırlanmıştır.

## ⚠️ Önemli Uyarı

Bu bir **beta sürümüdür**. Windows desteği test aşamasındadır ve bazı sorunlar içerebilir.

## Gereksinimler

- Windows 10/11 (64-bit)
- Yönetici hakları
- Wintun driver

## Adım 1: Wintun Driver Kurulumu

Karadul Windows'ta çalışmak için [Wintun](https://www.wintun.net/) driver'ına ihtiyaç duyar.

### Otomatik Kurulum (Önerilen)

PowerShell'i yönetici olarak çalıştırın:

```powershell
# Wintun indir
$url = "https://www.wintun.net/builds/wintun-0.14.1-amd64.zip"
$output = "$env:TEMP\wintun.zip"
Invoke-WebRequest -Uri $url -OutFile $output

# Zip'i aç
Expand-Archive -Path $output -DestinationPath "$env:TEMP\wintun" -Force

# System32'ye kopyala
Copy-Item "$env:TEMP\wintun\wintun.dll" -Destination "C:\Windows\System32\" -Force

Write-Host "Wintun kurulumu tamamlandi!"
```

### Manuel Kurulum

1. https://www.wintun.net/ adresine gidin
2. `wintun-0.14.1-amd64.zip` dosyasını indirin
3. Zip'ten `wintun.dll` dosyasını çıkarın
4. Aşağıdaki konumlardan birine kopyalayın:
   - `C:\Windows\System32\` (önerilen)
   - `karadul.exe` ile aynı dizin
   - Mevcut çalışma dizini

## Adım 2: Karadul'u İndirin

```powershell
# Binary indir
$url = "https://github.com/karadul/karadul/releases/download/v0.1.0-beta.1/karadul-windows-amd64.exe"
$output = "$env:USERPROFILE\karadul.exe"
Invoke-WebRequest -Uri $url -OutFile $output

# PATH'e ekle (opsiyonel)
$path = [Environment]::GetEnvironmentVariable("PATH", "User")
[Environment]::SetEnvironmentVariable("PATH", "$path;$env:USERPROFILE", "User")
```

## Adım 3: Kontrol

```powershell
# Versiyon kontrolü
karadul version

# Wintun kontrolü
karadul wintun-check
```

**Beklenen çıktı:**
```
✅ Wintun driver found
   Path: C:\Windows\System32\wintun.dll
   Checking DLL load...
✅ Wintun is ready to use
```

## Adım 4: Firewall Ayarları

```powershell
# Yönetici olarak PowerShell
karadul firewall setup
```

**Beklenen çıktı:**
```
✅ Firewall rules added successfully
```

Kontrol:
```powershell
karadul firewall check
```

## Adım 5: Mesh'e Katılma

### Koordinasyon Sunucusu

```powershell
# Sunucu başlat (opsiyonel - kendi sunucunuzu kurun)
karadul server --addr=:8080
```

### Node Olarak Katılma

```powershell
# Auth key oluştur (sunucuda)
karadul auth create-key

# Node başlat
karadul up --server=http://SUNUCU_IP:8080 --auth-key=KEY
```

## Sorun Giderme

### "Wintun driver not found" Hatası

```powershell
# Wintun DLL konumunu kontrol et
Get-ChildItem C:\Windows\System32\wintun.dll
Get-ChildItem $pwd\wintun.dll

# Eksikse yeniden kurulum yap
```

### "Access denied" Hatası

PowerShell'i **Yönetici olarak** çalıştırın.

### TUN Device Oluşturma Hatası

1. Mevcut TUN adaptörlerini kontrol edin
2. Zaten varsa farklı isim deneyin
3. Yönetici haklarını kontrol edin

```powershell
# Mevcut adaptörleri listele
Get-NetAdapter | Where-Object { $_.InterfaceDescription -like "*Wintun*" }
```

### Firewall Bloklama

```powershell
# Mevcut kuralları kontrol et
netsh advfirewall firewall show rule name="Karadul"

# Tekrar ekle
karadul firewall remove
karadul firewall setup
```

## Log ve Debug

```powershell
# Detaylı log
karadul up --server=http://... --log-level=debug

# Event Viewer'da kontrol
Get-WinEvent -FilterHashtable @{LogName='Application'; ProviderName='Karadul'}
```

## Geri Bildirim

Sorun yaşarsanız lütfen bildirin:

1. GitHub Issues: https://github.com/karadul/karadul/issues
2. Hata raporu şablonu:
   ```
   - Windows sürümü:
   - Karadul versiyonu:
   - Wintun konumu:
   - Hata mesajı:
   - Adımlar:
   ```

## Kaldırma

```powershell
# Firewall kurallarını kaldır
karadul firewall remove

# Binary'yi sil
Remove-Item $env:USERPROFILE\karadul.exe

# Wintun'u kaldır (isteğe bağlı)
Remove-Item C:\Windows\System32\wintun.dll
```

## Yararlı Komutlar

```powershell
# Versiyon kontrolü
karadul version

# Durum görüntüleme
karadul status

# Peer listesi
karadul peers

# Firewall kontrol
karadul firewall check

# Port izni ekle
karadul firewall allow-port 51820 udp
```

## Kaynaklar

- [Ana Depo](https://github.com/karadul/karadul)
- [Wintun](https://www.wintun.net/)
- [Sorun Bildir](../../issues)
