// topup.js — naikkan stok semua produk aktif jadi 10000 sebelum uji
// checkout, agar hasil mengukur throughput tulis murni (bukan kehabisan
// stok). Database dikembalikan dari backup setelah benchmark selesai.
const token = process.argv[2];
const API = 'http://127.0.0.1:8080/api/v1';

async function main() {
  const res = await fetch(`${API}/products?limit=100`);
  const env = await res.json();
  const products = env.data || [];
  let ok = 0;
  for (const p of products) {
    const body = {
      category_id: p.category_id,
      name: p.name,
      slug: p.slug,
      description: p.description || '',
      price: p.price,
      cost_price: p.cost_price ?? 0,
      stock: 10000,
      sku: p.sku,
      image_url: p.image_url || '',
      weight_gram: p.weight_gram || 0,
      is_active: true,
    };
    const r = await fetch(`${API}/products/${p.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });
    if (r.ok) ok++;
    else console.log('gagal topup', p.sku, r.status);
  }
  console.log(`topup stok: ${ok}/${products.length} produk -> 10000`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
