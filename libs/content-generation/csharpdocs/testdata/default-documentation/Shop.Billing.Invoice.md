#### [Shop](index.md 'index')
### [Shop\.Billing](Shop.Billing.md 'Shop\.Billing')

## Invoice Class

An invoice\.

```csharp
public record Invoice : System.IEquatable<Shop.Billing.Invoice>
```

Inheritance [System\.Object](https://learn.microsoft.com/en-us/dotnet/api/system.object 'System\.Object') → Invoice

Implements [System\.IEquatable&lt;](https://learn.microsoft.com/en-us/dotnet/api/system.iequatable-1 'System\.IEquatable\`1')[Invoice](Shop.Billing.Invoice.md 'Shop\.Billing\.Invoice')[&gt;](https://learn.microsoft.com/en-us/dotnet/api/system.iequatable-1 'System\.IEquatable\`1')
### Constructors

<a name='Shop.Billing.Invoice.Invoice(int)'></a>

## Invoice\(int\) Constructor

An invoice\.

```csharp
public Invoice(int OrderId);
```
#### Parameters

<a name='Shop.Billing.Invoice.Invoice(int).OrderId'></a>

`OrderId` [System\.Int32](https://learn.microsoft.com/en-us/dotnet/api/system.int32 'System\.Int32')
### Methods

<a name='Shop.Billing.Invoice.Issue()'></a>

## Invoice\.Issue\(\) Method

Issues it\.

```csharp
public string Issue();
```

#### Returns
[System\.String](https://learn.microsoft.com/en-us/dotnet/api/system.string 'System\.String')