package utils

import "strings"

// NormalizePhone menormalisasi nomor WhatsApp pelanggan ke format kanonis
// 62xxx (dipakai sebagai kunci loyalty agar 08xx dan 628xx jadi satu
// pelanggan yang sama):
//   "081234567890"  -> "6281234567890"
//   "+628123456789" -> "628123456789"
//   "628123456789"  -> "628123456789"
//   "81234567890"   -> "6281234567890"
// Karakter non-digit (spasi, -, +) dibuang. String kosong -> "".
func NormalizePhone(raw string) string {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	if d == "" {
		return ""
	}
	if strings.HasPrefix(d, "62") {
		return d
	}
	if strings.HasPrefix(d, "0") {
		return "62" + d[1:]
	}
	return "62" + d
}
