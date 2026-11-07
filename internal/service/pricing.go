package service

import (
	"encoding/json"
	"errors"
	"strings"

	"ecommerce-api/internal/domain"
)

// ApplyModifiers menghitung harga & HPP satuan sebuah item dari harga dasar
// produk ditambah delta semua varian yang dipilih pelanggan.
// Menolak delta negatif (tidak boleh ada modifier "diskon" dari klien).
func ApplyModifiers(basePrice, baseCost int64, mods []domain.SelectedModifier) (unitPrice, unitCost int64, err error) {
	unitPrice, unitCost = basePrice, baseCost
	for _, m := range mods {
		if strings.TrimSpace(m.Name) == "" {
			return 0, 0, errors.New("nama modifier tidak boleh kosong")
		}
		if m.PriceDelta < 0 || m.CostDelta < 0 {
			return 0, 0, errors.New("price_delta/cost_delta modifier tidak boleh negatif")
		}
		unitPrice += m.PriceDelta
		unitCost += m.CostDelta
	}
	return unitPrice, unitCost, nil
}

// MarshalSelectedModifiers mengubah daftar varian terpilih menjadi RawJSON
// untuk disimpan di OrderItem.SelectedModifiers (nil bila tidak ada).
func MarshalSelectedModifiers(mods []domain.SelectedModifier) (domain.RawJSON, error) {
	if len(mods) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(mods)
	if err != nil {
		return nil, err
	}
	return domain.RawJSON(b), nil
}
