<a id="shop"></a>

# shop

Shop: a tiny store.

<a id="shop.billing.invoice"></a>

# shop.billing.invoice

Invoices.

<a id="shop.billing.invoice.issue"></a>

#### issue

```python
def issue(order_id: int) -> str
```

Issue an invoice for an order.

<a id="shop.billing"></a>

# shop.billing

Billing.

<a id="shop.cart"></a>

# shop.cart

The shopping cart.

<a id="shop.cart.Cart"></a>

## Cart Objects

```python
class Cart()
```

A cart of items.

**Arguments**:

- `owner` - who owns it.

<a id="shop.cart.Cart.add"></a>

#### add

```python
def add(sku: str, quantity: int = 1) -> int
```

Add an item.

**Arguments**:

- `sku` - the product code.
- `quantity` - how many.
  

**Returns**:

  The new item count.

<a id="shop.cart.total"></a>

#### total

```python
def total(prices: list) -> float
```

Sum the prices.

