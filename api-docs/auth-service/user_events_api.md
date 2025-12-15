# User Events API Guide

## Summary
- `UserEventsController` exposes read/report capabilities for authenticated users’ security events, returning resource-wrapped payloads for consistency with the rest of the API.```15:85:app/Http/Controllers/Api/V1/UserEventsController.php
class UserEventsController extends Controller
{
    /**
     * Get a paginated list of user events.
     *
     * @return \App\Http\Resources\UserEventResourceCollection
     */
    public function index()
    {
        return UserEventResource::collection(
            UserEvent::whereBelongsto(request()->user())->simplePaginate(10)
        );
    }

    /**
     * Get the details of a user event.
     *
     * @param UserEvent $userEvent
     * @return \App\Http\Resources\UserEventResource
     */
    public function show(UserEvent $userEvent)
    {
        return new UserEventResource($userEvent);
    }

    /**
     * Store a report for a user event.
     *
     * @param ReportEventRequest $request
     * @param UserEvent $userEvent
     * @return \App\Http\Resources\UserEventReportResource
     */
    public function store(ReportEventRequest $request, UserEvent $userEvent)
    {
        $report = $userEvent->report()->create([
            'suspecious_citizen' => $request->suspecious_citizen,
            'event_description' => $request->event_description
        ]);

        return new UserEventReportResource($report);
    }

    /**
     * Send a response to a user event report.
     *
     * @param Request $request
     * @param UserEvent $userEvent
     * @return \App\Http\Resources\UserEventReportResponseResource
     */
    public function sendResponse(Request $request, UserEvent $userEvent)
    {
        $request->validate(['response' => 'required|string|max:300']);

        $response = $userEvent->report->responses()->create([
            'responser_name' => $request->user()->name,
            'response' => $request->response,
        ]);

        $userEvent->report->update(['status' => 1]);
        return new UserEventReportResponseResource($response);
    }

    /**
     * Close a user event report.
     *
     * @param UserEvent $userEvent
     * @return \Illuminate\Http\Response
     */
    public function closeEventReport(UserEvent $userEvent)
    {
        $userEvent->report->update(['closed' => 1]);
        return response()->noContent();
    }
}
```
- All endpoints live inside the API v1 authenticated stack and inherit the `auth:sanctum`, `verified`, and `activity` middleware, ensuring only verified sessions can review or act on event reports.```88:230:routes/api.php
Route::middleware(['auth:sanctum', 'verified', 'activity'])->group(function () {
    // ...
    Route::scopeBindings()->group(function () {
        // ...
        Route::controller(UserEventsController::class)->as('user-events.')->prefix('events')->group(function () {
            Route::get('/', 'index')->name('index');
            Route::get('/{userEvent}', 'show')->name('show');
            Route::post('/report/{userEvent}', 'store');
            Route::post('/report/response/{userEvent}', 'sendResponse');
            Route::post('/report/close/{userEvent}', 'closeEventReport');
        });
    });
});
```
- Reporting requires the `ReportEventRequest` form request, while responses undergo inline validation to guard against empty payloads and enforce a 300-character cap.```24:29:app/Http/Requests/ReportEventRequest.php
return [
    'suspecious_citizen' => 'nullable|string|exists:users,code',
    'event_description' => 'required|string|max:500'
];
```
```62:69:app/Http/Controllers/Api/V1/UserEventsController.php
$request->validate(['response' => 'required|string|max:300']);

$response = $userEvent->report->responses()->create([
    'responser_name' => $request->user()->name,
    'response' => $request->response,
]);
```

## Route Registry
| Method | Path | Middleware | Controller action | Response |
| --- | --- | --- | --- | --- |
| GET | `/api/events` | `auth:sanctum`, `verified`, `activity` | `index` | `200 OK` with paginated `UserEventResource` items |
| GET | `/api/events/{userEvent}` | `auth:sanctum`, `verified`, `activity` | `show` | `200 OK` single `UserEventResource` with embedded report |
| POST | `/api/events/report/{userEvent}` | `auth:sanctum`, `verified`, `activity` | `store` | `201 Created` with `UserEventReportResource` |
| POST | `/api/events/report/response/{userEvent}` | `auth:sanctum`, `verified`, `activity` | `sendResponse` | `201 Created` with `UserEventReportResponseResource`; report status flips to 1 |
| POST | `/api/events/report/close/{userEvent}` | `auth:sanctum`, `verified`, `activity` | `closeEventReport` | `204 No Content`; report `closed` flag becomes 1 |

The routes are defined via a scoped binding group so implicit model binding respects parent context, and they share the `user-events` route name prefix for selective `routeIs` checks.```224:229:routes/api.php
Route::controller(UserEventsController::class)->as('user-events.')->prefix('events')->group(function () {
    Route::get('/', 'index')->name('index');
    Route::get('/{userEvent}', 'show')->name('show');
    Route::post('/report/{userEvent}', 'store');
    Route::post('/report/response/{userEvent}', 'sendResponse');
    Route::post('/report/close/{userEvent}', 'closeEventReport');
});
```

## Authentication & Activity Policy
- **Session requirements:** Every endpoint sits inside the `auth:sanctum`, `verified`, and `activity` middleware trio, so callers must present a valid Sanctum token tied to a verified account, and their activity is logged.```88:108:routes/api.php
Route::middleware(['auth:sanctum', 'verified', 'activity'])->group(function () {
    // ...
});
```
- **Scoped bindings:** Thanks to `Route::scopeBindings()`, Laravel only resolves `UserEvent` instances associated with the expected parent context, preventing cross-user access via ID tampering.```101:108:routes/api.php
Route::scopeBindings()->group(function () {
    Route::controller(UserEventsController::class)->as('user-events.')->prefix('events')->group(function () {
        // ...
    });
});
```
- **Ownership filtering:** The `index` action further narrows results by running `whereBelongsto(request()->user())`, so pagination lists only the authenticated user’s events even outside of explicit policies.```21:24:app/Http/Controllers/Api/V1/UserEventsController.php
return UserEventResource::collection(
    UserEvent::whereBelongsto(request()->user())->simplePaginate(10)
);
```

## Validation Rules
### Report submission (`POST /api/events/report/{userEvent}`)
- `suspecious_citizen` (optional): must be a string referencing an existing citizen code in `users.code`.
- `event_description` (required): string up to 500 characters describing the suspicious activity.
- Authorization is unconditional (`authorize()` returns `true`), with trust delegated to route middleware for access control.```14:29:app/Http/Requests/ReportEventRequest.php
public function authorize()
{
    return true;
}

public function rules()
{
    return [
        'suspecious_citizen' => 'nullable|string|exists:users,code',
        'event_description' => 'required|string|max:500'
    ];
}
```

### Report response (`POST /api/events/report/response/{userEvent}`)
- `response` (required): plain string response capped at 300 characters, validated inline before persistence.```62:69:app/Http/Controllers/Api/V1/UserEventsController.php
$request->validate(['response' => 'required|string|max:300']);

$response = $userEvent->report->responses()->create([
    'responser_name' => $request->user()->name,
    'response' => $request->response,
]);
```

## Response Contracts
### `UserEventResource`
- Includes the event metadata (`event`, `ip`, `device`, `status`) plus Jalali-formatted `date` and `time`.
- When accessed via the `user-events.show` route, the response embeds the associated `report` payload.```20:29:app/Http/Resources/UserEventResource.php
return [
    'id' => $this->id,
    'event' => $this->event,
    'ip' => $this->ip,
    'device' => $this->device,
    'status' => $this->status ? 'موفق' : 'ناموفق',
    'date' => Jalalian::forge($this->created_at)->format('Y/m/d'),
    'time' => Jalalian::forge($this->created_at)->format('H:m:s'),
    $this->mergeWhen(request()->routeIs('user-events.show'), [
        'report' => new UserEventReportResource($this->report),
    ])
];
```

### `UserEventReportResource`
- Surfaces report identifiers, the suspect citizen code, description, status flags, and Jalali timestamps.
- Attaches all recorded responses as `UserEventReportResponseResource` entries.```18:27:app/Http/Resources/UserEventReportResource.php
return [
    'id' => $this->id,
    'suspecious_citizen' => $this->suspecious_citizen,
    'event_description' => $this->event_description,
    'status' => $this->status,
    'closed' => $this->closed,
    'date' => Jalalian::forge($this->created_at)->format('Y/m/d'),
    'time' => Jalalian::forge($this->created_at)->format('H:m:s'),
    'responses' => UserEventReportResponseResource::collection($this->responses)
];
```

### `UserEventReportResponseResource`
- Returns responder name, message body, and Jalali `date`/`time` stamps for each follow-up.```18:24:app/Http/Resources/UserEventReportResponseResource.php
return [
    'id' => $this->id,
    'responser_name' => $this->responser_name,
    'response' => $this->response,
    'date' => Jalalian::forge($this->created_at)->format('Y/m/d'),
    'time' => Jalalian::forge($this->created_at)->format('H:m:s'),
];
```

## Operational Notes
- `simplePaginate(10)` keeps the event listing lightweight for mobile clients by omitting total counts.```21:24:app/Http/Controllers/Api/V1/UserEventsController.php
return UserEventResource::collection(
    UserEvent::whereBelongsto(request()->user())->simplePaginate(10)
);
```
- Reporting responses both log the responder’s display name and mark the report as handled (`status` set to `1`), while closing the report flips the `closed` flag for downstream filters.```66:72:app/Http/Controllers/Api/V1/UserEventsController.php
$response = $userEvent->report->responses()->create([
    'responser_name' => $request->user()->name,
    'response' => $request->response,
]);

$userEvent->report->update(['status' => 1]);
```
```83:84:app/Http/Controllers/Api/V1/UserEventsController.php
$userEvent->report->update(['closed' => 1]);
return response()->noContent();
```
- Underlying models (`UserEvent`, `UserEventReport`, `UserEventReportResponse`) define the fillable attributes and relationships leveraged by the controller, ensuring mass-assignment safety and eager-loading compatibility.```13:28:app/Models/User/UserEvent.php
protected $fillable = [
    'user_id',
    'event',
    'ip',
    'device',
    'status',
];

public function report()
{
    return $this->hasOne(UserEventReport::class);
}
```
```12:28:app/Models/User/UserEventReport.php
protected $fillable = [
    'user_event_id',
    'suspecious_citizen',
    'event_description',
    'status',
    'closed'
];

public function responses()
{
    return $this->hasMany(UserEventReportResponse::class);
}
```
```12:20:app/Models/User/UserEventReportResponse.php
protected $fillable = [
    'user_event_report_id',
    'response',
    'responser_name',
];

public function report()
{
    return $this->belongsTo(UserEventReport::class);
}
```

