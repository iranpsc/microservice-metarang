# Order & Parsian Callback API

## Overview
- Centralized flow for purchasing virtual assets and routing Parsian payment callbacks.
- Backed by `App\Http\Controllers\Api\V1\OrderController`.
- Persists orders in `App\Models\Order` with linked `Transaction` and `Payment` records, then updates user wallets and referral bonuses.

## Authentication & Access Control
- `POST /api/order` sits inside the `auth:sanctum`, `verified`, `activity` middleware stack.
- Controller invokes `authorize('buyFromStore', User::class)`; see policy constraints below.
- Callback endpoint `/api/parsian/callback` is public so Parsian can reach it.

## Policy Summary
- `UserPolicy::buyFromStore(User $user)` (in `app/Policies/UserPolicy.php`): blocks users under 18 unless `permissions` are verified and the `BFR` flag is set; adults pass automatically.
- `OrderPolicy::canGetBonus(User $user, Order $order)` (in `app/Policies/OrderPolicy.php`): returns `true` only when the user has never logged a `firstOrder` record and the asset is not `irr`. Controls bonus crediting during callback handling.

## Validation Rules
- Handled by `App\Http\Requests\BuyAssetRequest`.
- Fields:
  - `amount`: required, numeric integer, minimum `1`.
  - `asset`: required, must be one of the enum values in `App\Enums\AssetTypes` (`psc`, `irr`, `red`, `blue`, `yellow`).
- Failing validation returns HTTP `422` with standard Laravel error payloads.

## Endpoint: POST /api/order
- **Purpose**: Create an order and obtain a Parsian payment URL.
- **Request body** (`application/json`):
  - `amount` – integer quantity of units to buy.
  - `asset` – enum string (`psc`, `irr`, `red`, `blue`, `yellow`).
- **Successful response** (`200 OK`):
  ```json
  {
    "link": "https://pec.shaparak.ir/NewIPG/?token=..."
  }
  ```
- **Workflow**:
  1. Policy gate (`buyFromStore`) ensures user eligibility.
  2. Determines conversion rate via `Variable::getRate($asset)`.
  3. Creates `Order` plus a morph-one `Transaction` (action=`deposit`).
  4. Selects Parsian merchant ID:
     - Standard: `config('parsian.merchant_id')` for non-`irr` assets.
     - Loan account: `config('parsian.loan_account_merchant_id')` for `irr`.
  5. Sends purchase request using `parsian()` SDK with callback URL `route('parsian.callback')`.
  6. On Parsian request failure, throws `ValidationException` → HTTP `422` containing Parsian error message under `error`.
  7. Stores returned Parsian `token` on the transaction and responds with the payment redirect link.
- **Side effects on success**: new order set to default status `-138`, pending Parsian verification.

## Endpoint: POST /api/parsian/callback
- **Purpose**: Receive Parsian gateway response and finalize the order.
- **Expected payload** (form-encoded query params from Parsian):
  - `OrderId` – the order primary key.
  - `status` – Parsian status code (`0` indicates success).
  - Additional gateway fields (e.g., `Token`, `RRN`, `CardMaskPan`, etc.) are proxied to the redirect URL.
- **Processing logic**:
  1. Fetches order with eager-loaded `user` and `transaction`. Missing records trigger `404`.
  2. When `status == 0`:
     - Calculates payment amount via `Variable::getRate`.
     - Selects merchant ID (same logic as order creation).
     - Calls Parsian verification API using the stored transaction token.
     - On verification success:
       - Updates order `status` to Parsian response status.
       - Updates transaction `status`, `ref_id`.
       - Evaluates `canGetBonus` policy; first-time, non-`irr` buyers receive:
         - `firstOrder` record saved with 50% bonus.
         - Wallet incremented by `amount + bonus`.
       - Otherwise wallet increments by `amount`.
       - Creates related `Payment` record recording `ref_id`, `card_pan`, `gateway=parsian`, `amount`, `product`.
       - Triggers referral logic (`ReferralService::referral`) for non-`irr` assets.
       - Dispatches `TransactionNotification` and calls `$user->deposit()` hook.
  3. When `status != 0`, marks order and transaction with the received status without verification.
  4. Redirects user (HTTP `302`) to `https://rgb.irpsc.com/metaverse/payment/verify?{original-query-string}` so the frontend can show the result.
- **Failure handling**:
  - Verification failure keeps order/transaction at previous status and still redirects with Parsian query parameters.
  - Missing or tampered `OrderId` yields Laravel `404`.

## Order Lifecycle & Status Codes
- Orders start with `status = -138` (default attribute in `App\Models\Order`).
- Successful verification replaces status with Parsian response status (usually `0`).
- Failed attempts persist the gateway-provided status code for audit.
- Linked `Transaction` mirrors order status and stores the Parsian `token` and `ref_id`.

## Parsian Configuration
- Env-driven settings in `config/parsian.php`:
  - `PARSIAN_MERCHANT_ID` & `PARSIAN_PIN` for standard sales.
  - `PARSIAN_LOAN_ACCOUNT_MERCHANT_ID` & `PARSIAN_LOAN_ACCOUNT_PIN` for `irr` purchases.
  - `PARSIAN_CALLBACK_URL` is unused here because callback URL is explicitly set to the named route.
- Ensure these credentials are present in deployment environments; otherwise Parsian requests will fail with validation errors.

## Testing Notes
- Use authenticated Sanctum tokens for manual `POST /api/order` calls.
- Mock or intercept Parsian SDK responses in automated tests to simulate:
  - Successful payment (`status=0`, verification success).
  - Failed initial request (forcing `ValidationException`).
  - Non-zero callback statuses (e.g., user cancellation).
- Verify wallet increments and `firstOrder` bonus records when policies allow.


