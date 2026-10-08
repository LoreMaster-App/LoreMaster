# Class: Store\<T\>

Keeps key/value pairs in memory.

## Example

```ts
const store = new Store<string>({ capacity: 10 })
store.set('a', '1')
```

## Type Parameters

### T

`T`

the type of the stored values

## Constructors

### Constructor

> **new Store**\<`T`\>(`options?`): `Store`\<`T`\>

Creates an empty store.

#### Parameters

##### options?

[`StoreOptions`](../interfaces/StoreOptions.md) = `{}`

#### Returns

`Store`\<`T`\>

## Accessors

### size

#### Get Signature

> **get** **size**(): `number`

How many entries are stored.

##### Returns

`number`

## Methods

### get()

> **get**(`key`): `T` \| `undefined`

Returns the value for `key`.

#### Parameters

##### key

`string`

the key to look up

#### Returns

`T` \| `undefined`

the value, or `undefined` when there is none

***

### set()

> **set**(`key`, `value`): `void`

Stores `value` under `key`.

#### Parameters

##### key

`string`

##### value

`T`

#### Returns

`void`
