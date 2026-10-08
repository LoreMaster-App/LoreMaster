#### [Shop](index.md 'index')
### [Shop](Shop.md 'Shop')

## Cart Class

A cart of items\.

```csharp
public class Cart
```

Inheritance [System\.Object](https://learn.microsoft.com/en-us/dotnet/api/system.object 'System\.Object') → Cart
### Constructors

<a name='Shop.Cart.Cart(string)'></a>

## Cart\(string\) Constructor

Creates a cart\.

```csharp
public Cart(string owner);
```
#### Parameters

<a name='Shop.Cart.Cart(string).owner'></a>

`owner` [System\.String](https://learn.microsoft.com/en-us/dotnet/api/system.string 'System\.String')

The owner\.
### Properties

<a name='Shop.Cart.Owner'></a>

## Cart\.Owner Property

Who owns the cart\.

```csharp
public string Owner { get; }
```

#### Property Value
[System\.String](https://learn.microsoft.com/en-us/dotnet/api/system.string 'System\.String')
### Methods

<a name='Shop.Cart.Add(string,int)'></a>

## Cart\.Add\(string, int\) Method

Adds an item\.

```csharp
public int Add(string sku, int quantity=1);
```
#### Parameters

<a name='Shop.Cart.Add(string,int).sku'></a>

`sku` [System\.String](https://learn.microsoft.com/en-us/dotnet/api/system.string 'System\.String')

The product code\.

<a name='Shop.Cart.Add(string,int).quantity'></a>

`quantity` [System\.Int32](https://learn.microsoft.com/en-us/dotnet/api/system.int32 'System\.Int32')

How many\.

#### Returns
[System\.Int32](https://learn.microsoft.com/en-us/dotnet/api/system.int32 'System\.Int32')  
The new item count\.