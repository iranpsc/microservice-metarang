# Notifications API Documentation

This document describes the Notifications API endpoints for frontend developers.

## Base URL
All endpoints are prefixed with `/api/notifications`

## Authentication
All endpoints require authentication. Include the authentication token in the request headers:
```
Authorization: Bearer {your-token}
```

---

## Endpoints

### 1. Get Unread Notifications

Retrieve all unread notifications for the authenticated user.

**Endpoint:** `GET /api/notifications`

**Headers:**
```
Authorization: Bearer {token}
```

**Response:** `200 OK`

**Response Body:**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "data": {
      "related-to": "transactions",
      "sender-name": "متارنگ",
      "sender-image": "https://example.com/uploads/img/logo.png",
      "message": "مقدار 100 PSC به حساب شما واریز گردید!"
    },
    "read_at": null,
    "date": "1403/09/15",
    "time": "14:30:25"
  },
  {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "data": {
      "related-to": "dynasty",
      "sender-name": "متارنگ",
      "sender-image": "https://example.com/uploads/img/logo.png",
      "message": "سلسله شما تاسیس شد."
    },
    "read_at": null,
    "date": "1403/09/14",
    "time": "10:15:30"
  }
]
```

**Response Fields:**
- `id` (string, UUID): Unique identifier for the notification
- `data` (object): Notification payload containing:
  - `related-to` (string): Category of the notification (e.g., "transactions", "dynasty", "trades")
  - `sender-name` (string): Name of the notification sender
  - `sender-image` (string, URL): URL to the sender's image/logo
  - `message` (string): The notification message text
- `read_at` (string|null): Timestamp when the notification was read, or `null` if unread
- `date` (string): Creation date in Jalali format (Y/m/d)
- `time` (string): Creation time in format (H:m:s)

**Example Request:**
```javascript
fetch('/api/notifications', {
  method: 'GET',
  headers: {
    'Authorization': 'Bearer your-token-here',
    'Content-Type': 'application/json'
  }
})
.then(response => response.json())
.then(data => console.log(data));
```

---

### 2. Get Single Notification

Retrieve details of a specific notification by ID.

**Endpoint:** `GET /api/notifications/{notification}`

**Path Parameters:**
- `notification` (string, UUID): The notification ID

**Headers:**
```
Authorization: Bearer {token}
```

**Response:** `200 OK`

**Response Body:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "related-to": "transactions",
    "sender-name": "متارنگ",
    "sender-image": "https://example.com/uploads/img/logo.png",
    "message": "مقدار 100 PSC به حساب شما واریز گردید!"
  },
  "read_at": null,
  "date": "1403/09/15",
  "time": "14:30:25"
}
```

**Error Responses:**
- `404 Not Found`: Notification not found

**Example Request:**
```javascript
const notificationId = '550e8400-e29b-41d4-a716-446655440000';
fetch(`/api/notifications/${notificationId}`, {
  method: 'GET',
  headers: {
    'Authorization': 'Bearer your-token-here',
    'Content-Type': 'application/json'
  }
})
.then(response => response.json())
.then(data => console.log(data));
```

---

### 3. Mark Notification as Read

Mark a specific notification as read.

**Endpoint:** `POST /api/notifications/read/{notification}`

**Path Parameters:**
- `notification` (string, UUID): The notification ID to mark as read

**Headers:**
```
Authorization: Bearer {token}
```

**Request Body:** None required

**Response:** `204 No Content`

**Error Responses:**
- `404 Not Found`: Notification not found

**Example Request:**
```javascript
const notificationId = '550e8400-e29b-41d4-a716-446655440000';
fetch(`/api/notifications/read/${notificationId}`, {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer your-token-here',
    'Content-Type': 'application/json'
  }
})
.then(response => {
  if (response.status === 204) {
    console.log('Notification marked as read');
  }
});
```

---

### 4. Mark All Notifications as Read

Mark all unread notifications for the authenticated user as read.

**Endpoint:** `POST /api/notifications/read/all`

**Headers:**
```
Authorization: Bearer {token}
```

**Request Body:** None required

**Response:** `204 No Content`

**Example Request:**
```javascript
fetch('/api/notifications/read/all', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer your-token-here',
    'Content-Type': 'application/json'
  }
})
.then(response => {
  if (response.status === 204) {
    console.log('All notifications marked as read');
  }
});
```

---

## Notification Data Structure

The `data` field in notification responses contains different properties depending on the notification type. Common fields include:

- `related-to` (string): Category identifier. Common values:
  - `"transactions"`: Financial transactions
  - `"dynasty"`: Dynasty-related notifications
  - `"trades"`: Trading/buying requests
- `sender-name` (string): Display name of the notification sender
- `sender-image` (string, URL): URL to the sender's image
- `message` (string): Human-readable notification message

**Note:** The `data` structure may vary based on the notification type. Always check the `related-to` field to determine how to handle the notification data.

---

## Date and Time Format

- **Date Format:** Jalali calendar format `Y/m/d` (e.g., "1403/09/15")
- **Time Format:** 24-hour format `H:m:s` (e.g., "14:30:25")

---

## Error Handling

All endpoints may return the following error responses:

- `401 Unauthorized`: Missing or invalid authentication token
- `404 Not Found`: Resource not found (for single notification endpoints)
- `500 Internal Server Error`: Server error

**Error Response Format:**
```json
{
  "message": "Error message here"
}
```

---

## Usage Examples

### Fetch and Display Notifications
```javascript
async function fetchNotifications() {
  try {
    const response = await fetch('/api/notifications', {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch notifications');
    }
    
    const notifications = await response.json();
    return notifications;
  } catch (error) {
    console.error('Error fetching notifications:', error);
    return [];
  }
}
```

### Mark Notification as Read
```javascript
async function markAsRead(notificationId) {
  try {
    const response = await fetch(`/api/notifications/read/${notificationId}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    return response.status === 204;
  } catch (error) {
    console.error('Error marking notification as read:', error);
    return false;
  }
}
```

### Mark All as Read
```javascript
async function markAllAsRead() {
  try {
    const response = await fetch('/api/notifications/read/all', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    return response.status === 204;
  } catch (error) {
    console.error('Error marking all notifications as read:', error);
    return false;
  }
}
```

---

## Notes

- Only unread notifications are returned by the `GET /api/notifications` endpoint
- After marking a notification as read, it will no longer appear in the unread notifications list
- Date and time values are formatted according to the Jalali calendar system
- The `read_at` field will be `null` for unread notifications and contain a timestamp for read notifications
