<!-- file: README.md -->
# API reference

| API | Version | Source |
|---|---|---|
| [Orders](orders/README.md) | 0.1.0 | `orders/openapi.json` |
| [Petstore](petstore/README.md) | 1.2.0 | `petstore.openapi.yaml` |

<!-- file: orders/README.md -->
# Orders

Version 0.1.0, OpenAPI 3.1.0. Source: `orders/openapi.json`.

## Operations

| Method | Path | Summary | Group |
|---|---|---|---|
| POST | `/orders` | Place an order | [orders](orders.md) |

The data types are described on the [schemas](schemas.md) page.

<!-- file: orders/orders.md -->
# Orders: orders

## POST /orders

**Place an order**

### Request body

Required.

| Content type | Schema |
|---|---|
| `application/json` | [Order](schemas.md) |

### Responses

| Status | Description | Content |
|---|---|---|
| `202` | Accepted. |  |

<!-- file: orders/schemas.md -->
# Orders: Schemas

## Order

Type: object

| Property | Type | Required | Description |
|---|---|---|---|
| `sku` | string | yes |  |
| `quantity` | integer or null | no | Default: 1. |

<!-- file: petstore/README.md -->
# Petstore

Version 1.2.0, OpenAPI 3.0.3. Source: `petstore.openapi.yaml`.

A sample API for managing pets.

Authenticate with an API key.

## Servers

| URL | Description |
|---|---|
| `https://api.example.com/v1` | Production |
| `https://staging.example.com/v1` |  |

## Operations

| Method | Path | Summary | Group |
|---|---|---|---|
| GET | `/pets` | List pets | [pets](pets.md) |
| POST | `/pets` | Add a pet | [pets](pets.md) |
| GET | `/pets/{petId}` | Get a pet | [pets](pets.md) |
| DELETE | `/pets/{petId}` | Remove a pet | [pets](pets.md) |
| GET | `/health` | Is it up? | [default](default.md) |

The data types are described on the [schemas](schemas.md) page.

<!-- file: petstore/pets.md -->
# Petstore: pets

Everything about pets.

## GET /pets

**List pets** · operation `listPets`

Returns every pet, newest first.

### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `limit` | query | integer (int32) | no | How many pets to return. Default: 20. |
| `status` | query | string | no | Only pets with this status. One of: `available`, `pending`, `sold`. |

### Responses

| Status | Description | Content |
|---|---|---|
| `200` | A page of pets. | `application/json`: array of [Pet](schemas.md) |
| `default` | Something went wrong. | `application/problem+json`: [Problem](schemas.md) |

## POST /pets

**Add a pet**

### Request body

Required. The pet to add.

| Content type | Schema |
|---|---|
| `application/json` | [NewPet](schemas.md) |
| `application/xml` | [NewPet](schemas.md) |

### Responses

| Status | Description | Content |
|---|---|---|
| `201` | Created. | `application/json`: [Pet](schemas.md) |

## GET /pets/{petId}

**Get a pet**

### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `petId` | path | string (uuid) | yes | The pet's id. |

### Responses

| Status | Description | Content |
|---|---|---|
| `200` | The pet. | `application/json`: [Pet](schemas.md) |
| `404` | No such pet. |  |

## DELETE /pets/{petId}

**Remove a pet** · **deprecated**

### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `petId` | path | string | yes | Overrides the path-level description. |

### Responses

| Status | Description | Content |
|---|---|---|
| `204` | Removed. |  |

<!-- file: petstore/default.md -->
# Petstore: default

## GET /health

**Is it up?**

### Responses

| Status | Description | Content |
|---|---|---|
| `200` | Yes. |  |

<!-- file: petstore/schemas.md -->
# Petstore: Schemas

## NewPet

Type: object

| Property | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | What the pet is called. |
| `tag` | string or null | no |  |

## Pet

A pet that lives here.

Type: all of [NewPet](schemas.md), object

| Property | Type | Required | Description |
|---|---|---|---|
| `id` | string (uuid) | yes |  |
| `status` | string | no | One of: `available`, `pending`, `sold`. Default: available. |
| `owner` | string or null | no | The owner's name, if any. |
| `friends` | array of [Pet](schemas.md) | no |  |
| `pipe` | string | no | Contains a \| pipe and &lt;angle> brackets. |

## Problem

Type: object

| Property | Type | Required | Description |
|---|---|---|---|
| `title` | string | no |  |
| `detail` | string | no |  |

## Shape

Type: one of [Pet](schemas.md), [Problem](schemas.md)

