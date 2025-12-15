# Citizen API Guide

## Summary
- `PublicProfileController` serves the citizen-facing profile, referral listing, and referral analytics endpoints without requiring authentication while still respecting per-user privacy settings.```24:67:app/Http/Controllers/Api/V1/PublicProfileController.php
public function home(User $user)
{
    $user->load(['kyc', 'personalInfo', 'profilePhotos', 'settings:id,user_id,privacy']);
    return new PersonalInfo($user);
}
// ... existing code ...
```
- Referral pagination and charting logic live in `ReferralService`, which applies search filtering and Jalali-based aggregations to the caller’s downstream responses.```68:242:app/Services/ReferralService.php
public function getReferrals(Request $request, User $user)
{
    $query = $user->referrals()->with(['referrerOrders' => function ($query) {
        $query->latest();
    }, 'kyc:id,user_id,fname,lname', 'latestProfilePhoto'])
        ->orderBy(function ($query) {
            $query->select('created_at')
                ->from('referral_order_histories')
                ->whereColumn('referral_id', 'users.referrer_id')
                ->latest()
                ->limit(1);
        }, 'desc');
// ... existing code ...
```

## Authentication & Routing
- The three citizen endpoints are publicly reachable and mapped under `/api/citizen` in `routes/api.php`; route-model binding pulls citizens by `users.code`.```254:258:routes/api.php
Route::controller(PublicProfileController::class)->prefix('citizen')->group(function () {
    Route::get('/{user:code}', 'home');
    Route::get('/{user:code}/referrals', 'referrals');
    Route::get('/{user:code}/referrals/chart', 'referralChart');
});
```
- No middleware is applied at the group level, so callers do not need to authenticate. Any authorization constraints are enforced by resource privacy filters rather than policies.

## Policies & Data Exposure
- Field-level privacy honors each citizen’s `settings->privacy` flags; attributes only appear in the JSON payload when the boolean gate allows it.```18:135:app/Http/Resources/PublicProfile/PersonalInfo.php
return [
    'profilePhotos' => ProfilePhotoResource::collection($this->whenLoaded('profilePhotos')),
    'kyc' => $this->whenLoaded('kyc', function () {
        return [
            $this->mergeWhen($this->checkFilter('nationality'), [
                'nationality' => url('/uploads/flags/iran.svg'),
            ]),
// ... existing code ...
    $this->mergeWhen($this->checkFilter('score'), [
        'score' => $this->score,
    ]),
    'score_percentage_to_next_level' => getScorePercentageToNextLevel($this->latest_level, $this->score),
// ... existing code ...
];
```
- Referral responses expose the invitee’s display name, profile photo URL, and referral order history only when those relations are eagerly loaded.```15:35:app/Http/Resources/ReferralResource.php
return [
    'id' => $this->id,
    'code' => $this->code,
    'name' => $this->whenLoaded('kyc', function () {
        return $this->kyc->full_name;
    }) ?? $this->name,
    'image' => $this->whenLoaded('latestProfilePhoto', function () {
        return $this->latestProfilePhoto->url;
    }),
// ... existing code ...
];
```
- Because these endpoints are public, consider applying rate limiting at the gateway or upstream proxy to protect citizen privacy from enumeration attacks.

## Validation Rules
- `user:code` path parameters rely on implicit route-model binding; invalid codes surface as `404 Not Found`.
- `GET /api/citizen/{code}/referrals` accepts an optional `search` query. When present, it is used in a `LIKE` clause against the referral’s `name` and `code` fields; no additional sanitisation is performed.
- `GET /api/citizen/{code}/referrals/chart` reads an optional `range` query. Accepted values are `daily`, `weekly`, `monthly`, or `yearly`. Unknown values fall back to `daily`.```50:67:app/Http/Controllers/Api/V1/PublicProfileController.php
$range = $request->input('range', 'daily');
return match ($range) {
    'yearly' => response()->json(['data' => $this->referralService->getYearlyStats($referrals)]),
    'monthly' => response()->json(['data' => $this->referralService->getMonthlyStats($referrals)]),
    'weekly' => response()->json(['data' => $this->referralService->getWeeklyStats($referrals)]),
    'daily' => response()->json(['data' => $this->referralService->getDailyStats($referrals)]),
    default => response()->json(['data' => $this->referralService->getDailyStats($referrals)]),
};
```
- Pagination is fixed at 10 records per page via `simplePaginate(10)`; callers should follow the `next_page_url` cursor to continue.

## Endpoints

### GET `/api/citizen/{code}`
- **Purpose**: Returns the public profile for the citizen identified by `code`.
- **Path parameters**:
  - `code` — case-insensitive citizen identifier stored in `users.code`; must belong to an existing user or the request returns `404`.
- **Response body**:
  - `profilePhotos` — array of `{id, url}` pairs (filtered by privacy flags).
  - `kyc` — subset of KYC attributes (`nationality`, `fname`, `lname`, `birth_date`, `phone`, `email`, `address`) depending on the citizen’s privacy configuration.
  - `code`, `name`, `position`, `registered_at` — basic profile metadata when allowed.
  - `customs` — custom fields such as `occupation`, `education`, `prediction`, etc., plus a `passions` map where enabled entries resolve to hosted icon URLs.
  - `score`, `score_percentage_to_next_level` — numeric stats; `current_level` with level metadata and `achieved_levels` appears when the citizen has ranking data and has opted in.
  - `avatar` — static 3D avatar URL gated by privacy.
- **HTTP status**: `200 OK` on success; `404` when the citizen code is unknown.

### GET `/api/citizen/{code}/referrals`
- **Purpose**: Lists the first-level referrals made by the target citizen, ordered by the most recent referral-order activity.
- **Queries**:
  - `search` (optional string) — case-insensitive partial match against referral `name` or `code`.
- **Response body**:
  - `data` — array of referrals, each containing `id`, `code`, `name`, optional `image`, and `referrerOrders` history (each entry includes `id`, `amount`, `created_at` in `Y-m-d H:i:s` Jalali format).
  - `meta` — simple pagination cursors (`current_page`, `next_page_url`, etc.) from Laravel’s `simplePaginate`.
- **HTTP status**: `200 OK`.
- **Notes**: The endpoint does not require authentication but still respects referral privacy by reusing eager-loaded relations; empty arrays signal either no referrals or privacy restrictions.

### GET `/api/citizen/{code}/referrals/chart`
- **Purpose**: Provides aggregated referral analytics for the citizen across selectable time ranges.
- **Queries**:
  - `range` (optional string) — one of `daily`, `weekly`, `monthly`, `yearly`; defaults to `daily`.
- **Response body**:
  - `data.total_referrals_count` — stringified count of referrals inside the requested range.
  - `data.total_referral_orders_amount` — stringified sum of referral-order amounts for the range.
  - `data.chart_data` — time-bucketed collection (hours/days/months/years) including counts and total amounts per bucket.
- **HTTP status**: `200 OK`.
- **Notes**: Bucketing leverages Jalali calendar conversions for labels, ensuring locale-appropriate output. When no referrals exist in the chosen window, totals and chart rows return `0`.

## Operational Considerations
- Apply external caching when traffic patterns justify it; the controller currently reads directly from the database on every call.
- Because the endpoints skip authentication, pair them with rate limiting and anomaly detection upstream to mitigate scraping risks.
- Downstream consumers should normalise Jalali timestamps if Gregorian timelines are required.


