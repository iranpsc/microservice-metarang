# Store Packages API Guide

## Summary
- `POST /api/store` returns pricing and media metadata for multiple store packages identified by their option codes.
- The request must provide at least two package codes; codes are resolved against the `options` table and wrapped with `PackageResource`.
- `PackageResource` augments the raw option data with the latest asset exchange rate (`Variable::getRate`) and the associated image URL when present.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| POST | `/api/store` | – | `HomeController@getStorePackages` | Return details for a batch of store packages. |

The route is registered in `routes/api.php` under the `HomeController` route group and does not enforce authentication or rate limiting by default.

## Request Contract
- Body: JSON
  - `codes` (required `array`, min length 2) – List of option codes to retrieve.
  - `codes.*` (required `string`, min length 2) – Each entry must be a non-empty string code.

Validation is enforced by `HomeController@getStorePackages`; any violation yields a 422 JSON response containing the validation errors.

## Response Contract
- Status: 200
- Body: JSON array of package objects, each shaped by `PackageResource`.
  - `id` (`int`) – Primary key of the option record.
  - `code` (`string`) – Option code provided in the request.
  - `asset` (`string`) – Underlying asset symbol stored on the option.
  - `amount` (`numeric`) – Package amount configured for the option.
  - `unitPrice` (`float`) – Lookup from `Variable::getRate($asset)`; represents the current asset price.
  - `image` (`string|null`) – Absolute URL of the option image if the polymorphic `image` relation is populated.

Returned packages mirror the order yielded by the `Option::whereIn(...)->get()` query, which is database-dependent; clients should not assume the response preserves the request order without explicit ordering logic.

## Data Sources & Dependencies
- `options` table: must contain rows keyed by `code`, `asset`, and `amount`. Missing codes result in the corresponding package being omitted from the response.
- `variables` table: provides asset pricing for `unitPrice`; missing rows cause `Variable::getRate` to return `null`, propagating `null` to the response.
- `images` table: polymorphic relationship (`imageable_id`, `imageable_type`) supplies optional package thumbnails.

The endpoint executes a single `whereIn` query against `options`; there is no eager loading for images, so accessing `image` triggers an additional query per option unless global eager loading is configured.

## Error Modes
- 422 – Validation failure when `codes` is missing, not an array, or contains fewer than two valid strings.
- 200 with empty array – All provided codes fail to resolve against the `options` table.
- 500 – Unhandled exceptions (for example, database connection issues). The endpoint does not include explicit error handling beyond validation.

## Usage Recommendations
- Supply only existing option codes to avoid empty responses; the endpoint intentionally omits unknown codes instead of raising errors.
- If deterministic ordering matters, sort the response on the client or request only codes belonging to a single logical group maintained in the database with predictable ordering.
- Cache results when possible; package metadata and prices change infrequently compared to request volume.
- Combine with authentication middleware if package visibility becomes user-specific; the current route is public.


