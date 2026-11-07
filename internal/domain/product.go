package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Varian & add-on produk (modifiers).
//
// Format modifiers_json (kolom TEXT di Product), contoh untuk minuman:
//
//	[
//	  {"group":"Temperature","type":"single","options":[
//	    {"name":"Iced","price_delta":0,"cost_delta":0},
//	    {"name":"Hot","price_delta":0,"cost_delta":0}]},
//	  {"group":"Sugar Level","type":"single","options":[
//	    {"name":"Normal Sugar","price_delta":0,"cost_delta":0},
//	    {"name":"Less Sugar","price_delta":0,"cost_delta":0},
//	    {"name":"No Sugar","price_delta":0,"cost_delta":0}]},
//	  {"group":"Ice Level","type":"single","options":[
//	    {"name":"Normal Ice","price_delta":0,"cost_delta":0},
//	    {"name":"Less Ice","price_delta":0,"cost_delta":0}]},
//	  {"group":"Add-Ons","type":"multiple","options":[
//	    {"name":"Extra Espresso Shot","price_delta":5000,"cost_delta":2500},
//	    {"name":"Upgrade Oat Milk","price_delta":8000,"cost_delta":4500},
//	    {"name":"Extra Cheese / Sambal","price_delta":4000,"cost_delta":2000}]}
//	]
//
// Type "single" = pelanggan pilih tepat 1 opsi; "multiple" = boleh 0..n.
// price_delta/cost_delta dalam rupiah, ditambahkan ke harga/HPP dasar.
// Pilihan pelanggan per item order disimpan di OrderItem.SelectedModifiers
// sebagai array datar: [{"name":"Iced","price_delta":0,"cost_delta":0},
// {"name":"Extra Espresso Shot","price_delta":5000,"cost_delta":2500}].
// ---------------------------------------------------------------------------

// ModifierGroup adalah satu kelompok varian (mis. Temperature).
type ModifierGroup struct {
	Group   string           `json:"group"`
	Type    string           `json:"type"`
	Options []ModifierOption `json:"options"`
}

// ModifierOption adalah satu opsi varian/add-on.
type ModifierOption struct {
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
	CostDelta  int64  `json:"cost_delta"`
}

// SelectedModifier adalah satu varian yang dipilih pelanggan pada sebuah
// item order (format sama dengan ModifierOption, tanpa grup).
type SelectedModifier struct {
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
	CostDelta  int64  `json:"cost_delta"`
}

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Slug        string         `gorm:"size:120;uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"size:500" json:"description"`
	ImageURL    string         `gorm:"size:255" json:"image_url"`
	Products    []Product      `gorm:"foreignKey:CategoryID" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CategoryID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name        string         `gorm:"size:150;not null;index" json:"name"`
	Slug        string         `gorm:"size:170;uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	Price       int64          `gorm:"not null;check:price>=0" json:"price"`
	CostPrice   int64          `gorm:"not null;default:0;check:cost_price>=0" json:"cost_price"`
	Stock       int            `gorm:"not null;default:0;check:stock>=0" json:"stock"`
	SKU         string         `gorm:"size:50;uniqueIndex;not null" json:"sku"`
	ImageURL    string         `gorm:"size:255" json:"image_url"`
	WeightGram  int            `gorm:"not null;default:0" json:"weight_gram"`
	IsActive    bool           `gorm:"not null;default:true;index" json:"is_active"`
	// IsAvailable adalah toggle "Habis Hari Ini" (1-klik sold out) yang
	// tidak mengubah Stock. Produk is_available=false tidak bisa di-checkout.
	IsAvailable bool `gorm:"not null;default:true;index" json:"is_available"`
	// ModifiersJSON menyimpan daftar ModifierGroup (lihat komentar di atas).
	// Modifiers adalah bentuk ter-parse-nya, diisi otomatis via AfterFind.
	ModifiersJSON RawJSON         `gorm:"type:text;column:modifiers_json" json:"modifiers_json"`
	Modifiers     []ModifierGroup `gorm:"-" json:"modifiers,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	DeletedAt     gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// AfterFind mem-parse modifiers_json ke field transient Modifiers.
func (p *Product) AfterFind(tx *gorm.DB) error {
	p.Modifiers = nil
	if len(p.ModifiersJSON) == 0 {
		return nil
	}
	var groups []ModifierGroup
	if err := json.Unmarshal(p.ModifiersJSON, &groups); err != nil {
		return nil
	}
	p.Modifiers = groups
	return nil
}

// SetModifiers mengisi ModifiersJSON dari daftar grup (dipakai seeder &
// service saat menyimpan konfigurasi varian produk).
func (p *Product) SetModifiers(groups []ModifierGroup) error {
	if len(groups) == 0 {
		p.ModifiersJSON = nil
		p.Modifiers = nil
		return nil
	}
	b, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	p.ModifiersJSON = b
	p.Modifiers = groups
	return nil
}
