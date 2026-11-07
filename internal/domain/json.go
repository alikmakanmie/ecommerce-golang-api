package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// RawJSON menyimpan JSON mentah sebagai kolom TEXT (SQLite/Postgres) dan
// di-encode apa adanya di respons API — bukan string JSON ganda.
// Dipakai untuk modifiers_json produk & selected_modifiers order item.
// Nilai kosong (nil/len 0) dibaca sebagai null di JSON dan disimpan NULL.
type RawJSON []byte

func (j RawJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *RawJSON) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = []byte(v)
		return nil
	default:
		return fmt.Errorf("RawJSON: tipe tidak didukung %T", value)
	}
}

func (j RawJSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *RawJSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("RawJSON: UnmarshalJSON pada pointer nil")
	}
	if string(data) == "null" {
		*j = nil
		return nil
	}
	// Validasi JSON valid sebelum disimpan.
	var tmp json.RawMessage
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	*j = append((*j)[:0], data...)
	return nil
}
